package compiler

import (
	"fmt"
	"go/ast"
	"go/token"

	"tgo/pkg/syntax"
)

type enumJSONUse struct {
	enum           bool
	external       bool
	adjacent       bool
	foldedAdjacent bool
}

type sourceModelConfig struct {
	jsonPackage     string
	jsonV2Package   string
	jsonTextPackage string
	stringsPackage  string
	fmtPackage      string
	externalJSONTo  string
	adjacentJSONTo  string
}

func enumJSONHelperNames(tree *syntax.File, used map[string]bool) (string, string) {
	for _, declaration := range tree.Declarations {
		node, ok := syntax.EnumDeclarationOf(declaration)
		if !ok {
			continue
		}
		return freshIdentifier("tgo"+node.Name.Name+"ExternalJSONTo", used),
			freshIdentifier("tgo"+node.Name.Name+"AdjacentJSONTo", used)
	}
	return "", ""
}

// sourceModels builds the projected model declarations and source edits.
func sourceModels(
	files *token.FileSet,
	file *token.File,
	tree *syntax.File,
	sourceName string,
	data []byte,
	erasedData []byte,
	config sourceModelConfig,
) ([]*model, []edit, enumJSONUse, int, error) {
	models := []*model(nil)
	edits := []edit(nil)
	use := enumJSONUse{}
	firstEnumEdit := -1
	for _, declaration := range tree.Declarations {
		item, replacement, itemUse, err := sourceModel(
			files, erasedData, declaration, sourceName, config,
		)
		if err != nil {
			return nil, nil, enumJSONUse{}, -1, err
		}
		use.enum = use.enum || itemUse.enum
		use.external = use.external || itemUse.external
		use.adjacent = use.adjacent || itemUse.adjacent
		use.foldedAdjacent = use.foldedAdjacent || itemUse.foldedAdjacent
		if item == nil {
			continue
		}
		startPositionValue := syntax.DeclarationPosition(declaration)
		endPositionValue := syntax.DeclarationEnd(declaration)
		if item.Enum && firstEnumEdit < 0 {
			firstEnumEdit = len(edits)
		}
		edits = append(edits, edit{
			start: file.Offset(startPositionValue),
			end:   declarationEditEnd(data, file.Offset(endPositionValue)),
			text: generatedSource(
				sourceName, files.Position(startPositionValue).Line,
				files.Position(endPositionValue).Line, replacement,
			),
			projectionOnly: plainStructProjection(item),
		})
		models = append(models, item)
	}
	return models, edits, use, firstEnumEdit, nil
}

// sourceModel builds one projected model declaration.
func sourceModel(
	files *token.FileSet,
	erasedData []byte,
	declaration *syntax.Declaration,
	sourceName string,
	config sourceModelConfig,
) (*model, string, enumJSONUse, error) {
	if node, ok := syntax.EnumDeclarationOf(declaration); ok {
		item := enumModel(files, erasedData, node)
		if err := validateEnumPublicNames(item, node); err != nil {
			return nil, "", enumJSONUse{}, fmt.Errorf(
				"%s: %w", files.Position(node.Name.Start), err,
			)
		}
		if err := configureEnumJSON(item, node); err != nil {
			return nil, "", enumJSONUse{}, fmt.Errorf(
				"%s: %w", files.Position(node.Name.Start), err,
			)
		}
		use := enumJSONUse{
			enum: true, external: item.JSON.Form == "external",
			adjacent: item.JSON.Form == "adjacent",
			foldedAdjacent: item.JSON.Form == "adjacent" &&
				jsonStructFieldName(item.JSON.Tag) &&
				jsonStructFieldName(item.JSON.Content),
		}
		replacement := enumGo(sourceName, item, config.fmtPackage) + enumJSONGo(
			item, config.jsonPackage, config.jsonV2Package,
			config.jsonTextPackage, config.stringsPackage, config.fmtPackage,
			config.externalJSONTo, config.adjacentJSONTo,
		)
		return item, replacement, use, nil
	}
	if node, ok := syntax.StructDeclarationOf(declaration); ok {
		item := structModel(files, erasedData, node)
		if item.CheckedStruct {
			if err := validateCheckedStructFields(files, node); err != nil {
				return nil, "", enumJSONUse{}, err
			}
			return item, checkedStructGo(sourceName, item), enumJSONUse{}, nil
		}
		return item, "type " + item.Name + " struct {\n" +
			fieldDecls(sourceName, item.Fields) + "}\n", enumJSONUse{}, nil
	}
	return nil, "", enumJSONUse{}, nil
}

func validateEnumPublicNames(declaration *model, node *syntax.EnumDeclaration) error {
	type generatedName struct {
		variant string
	}
	generated := map[string]generatedName{
		declaration.Name:         {},
		declaration.Name + "Tag": {},
	}
	for _, item := range node.Variants {
		names := []string{
			declaration.Name + item.Name.Name,
			declaration.Name + "Tag" + item.Name.Name,
			enumConstructorName(declaration.Name, item.Name.Name),
		}
		if len(item.Fields) > 0 {
			names = append(names, enumCarrierName(declaration.Name, item.Name.Name))
		}
		for _, name := range names {
			previous, exists := generated[name]
			if !exists {
				generated[name] = generatedName{variant: item.Name.Name}
				continue
			}
			if previous.variant != "" {
				return fmt.Errorf("enum variants %s and %s both generate %s",
					previous.variant, item.Name.Name, name)
			}
			return fmt.Errorf("enum variant %s generates %s, which conflicts with generated %s API",
				item.Name.Name, name, declaration.Name)
		}
	}
	return nil
}

func embeddedFieldName(expression *syntax.Expression) string {
	if expression == nil {
		return ""
	}
	if expression.Tag() == syntax.ExpressionTagIdentifier {
		return expression.IdentifierPayload().Value.Name
	}
	if expression.Tag() == syntax.ExpressionTagSelector {
		return expression.SelectorPayload().Value.Selector.Name
	}
	if expression.Tag() == syntax.ExpressionTagStar {
		return embeddedFieldName(expression.StarPayload().Value.Expression)
	}
	if expression.Tag() == syntax.ExpressionTagNonNilPointer {
		return embeddedFieldName(expression.NonNilPointerPayload().Value.Type)
	}
	if expression.Tag() == syntax.ExpressionTagParenthesized {
		return embeddedFieldName(expression.ParenthesizedPayload().Value.Expression)
	}
	if expression.Tag() == syntax.ExpressionTagIndex {
		return embeddedFieldName(expression.IndexPayload().Value.Expression)
	}
	if expression.Tag() == syntax.ExpressionTagIndexList {
		return embeddedFieldName(expression.IndexListPayload().Value.Expression)
	}
	return ""
}

func plainStructProjection(item *model) bool {
	if item.Enum || item.CheckedStruct {
		return false
	}
	for _, field := range item.Fields {
		if field.Default != "" {
			return false
		}
	}
	return true
}

func declarationEditEnd(data []byte, end int) int {
	cursor := end
	for cursor < len(data) && (data[cursor] == ' ' || data[cursor] == '\t') {
		cursor++
	}
	if cursor < len(data) && data[cursor] == ';' {
		return cursor + 1
	}
	return end
}

func enumModel(files *token.FileSet, data []byte, declaration *syntax.EnumDeclaration) *model {
	position := files.Position(declaration.Name.Start)
	result := &model{
		Name:     declaration.Name.Name,
		Enum:     true,
		Line:     position.Line,
		Column:   position.Column,
		Variants: nil,
		Fields:   nil,
	}
	for _, item := range declaration.Variants {
		fields := make([]field, 0, len(item.Fields))
		for _, itemField := range item.Fields {
			fields = append(fields, sourceModelField(files, data, itemField)...)
		}
		result.Variants = append(result.Variants, variant{
			Name:   item.Name.Name,
			Fields: fields,
		})
	}
	return result
}

func structModel(files *token.FileSet, data []byte, declaration *syntax.StructDeclaration) *model {
	position := files.Position(declaration.Name.Start)
	fields := make([]field, 0, len(declaration.Fields))
	for _, itemField := range declaration.Fields {
		fields = append(fields, sourceModelField(files, data, itemField)...)
	}
	return &model{
		Name:          declaration.Name.Name,
		Enum:          false,
		CheckedStruct: declaration.Checked != token.NoPos,
		Line:          position.Line,
		Column:        position.Column,
		Variants:      nil,
		Fields:        fields,
	}
}

func validateCheckedStructFields(
	files *token.FileSet,
	declaration *syntax.StructDeclaration,
) error {
	for _, field := range declaration.Fields {
		if len(field.Field.Names) == 0 {
			name := embeddedFieldName(field.Field.Type)
			if ast.IsExported(name) {
				return fmt.Errorf(
					"%s: checked struct field %s must be private",
					files.Position(syntax.ExpressionPosition(field.Field.Type)), name,
				)
			}
			continue
		}
		for _, name := range field.Field.Names {
			if ast.IsExported(name.Name) {
				return fmt.Errorf(
					"%s: checked struct field %s must be private",
					files.Position(name.Start), name.Name,
				)
			}
		}
	}
	return nil
}

func sourceModelField(files *token.FileSet, data []byte, declaration *syntax.TGoField) []field {
	result := []field(nil)
	typeText := sourceText(
		files, data, syntax.ExpressionPosition(declaration.Field.Type),
		syntax.ExpressionEnd(declaration.Field.Type),
	)
	typePosition := files.Position(syntax.ExpressionPosition(declaration.Field.Type))
	defaultText := ""
	defaultLine := 0
	defaultColumn := 0
	if declaration.Default != nil {
		defaultText = sourceText(
			files, data, syntax.ExpressionPosition(declaration.Default),
			syntax.ExpressionEnd(declaration.Default),
		)
		defaultPosition := files.Position(syntax.ExpressionPosition(declaration.Default))
		defaultLine = defaultPosition.Line
		defaultColumn = defaultPosition.Column
	}
	tag := ""
	if declaration.Field.Tag != nil {
		tag = declaration.Field.Tag.Value
	}
	if len(declaration.Field.Names) == 0 {
		return []field{newField(
			"", typeText, tag, defaultText, typePosition, defaultLine, defaultColumn,
		)}
	}
	for _, name := range declaration.Field.Names {
		result = append(result, newField(
			name.Name, typeText, tag, defaultText, typePosition, defaultLine, defaultColumn,
		))
	}
	return result
}

func newField(
	name string,
	typeText string,
	tag string,
	defaultText string,
	typePosition token.Position,
	defaultLine int,
	defaultColumn int,
) field {
	return field{
		Name:          name,
		Type:          typeText,
		Tag:           tag,
		Default:       defaultText,
		TypeLine:      typePosition.Line,
		TypeColumn:    typePosition.Column,
		DefaultLine:   defaultLine,
		DefaultColumn: defaultColumn,
	}
}

func sourceText(
	files *token.FileSet,
	data []byte,
	start token.Pos,
	end token.Pos,
) string {
	file := files.File(start)
	return string(data[file.Offset(start):file.Offset(end)])
}
