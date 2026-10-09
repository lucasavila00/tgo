package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"tgo/pkg/syntax"
)

// parseSource parses tgo syntax, lowers declarations, and parses the Go projection.
func parseSource(files *token.FileSet, name string, data []byte) (*source, error) {
	tree, err := syntax.ParseFile(
		files,
		name,
		data,
		syntax.ParseComments|syntax.AllErrors,
	)
	if err != nil {
		return nil, err
	}
	used := make(map[string]bool)
	syntax.Inspect(tree, func(node *syntax.Node) bool {
		if identifier, ok := syntax.IdentifierOf(node); ok {
			used[identifier.Name] = true
		}
		return true
	})
	defaultMarker := freshIdentifier("__tgo_defaults", used)
	jsonPackage := freshIdentifier("__tgo_json", used)
	jsonV2Package := freshIdentifier("__tgo_jsonv2", used)
	jsonTextPackage := freshIdentifier("__tgo_jsontext", used)
	stringsPackage := freshIdentifier("__tgo_strings", used)
	fmtPackage := freshIdentifier("__tgo_fmt", used)
	externalJSONTo, adjacentJSONTo := enumJSONHelperNames(tree, used)
	file := files.File(tree.Package)
	erasedData, nonNilLocations := eraseNonNilTypes(files, file, tree, data)
	models, edits, jsonUse, firstEnumEdit, err := sourceModels(
		files, file, tree, name, data, erasedData,
		sourceModelConfig{
			jsonPackage: jsonPackage, jsonV2Package: jsonV2Package,
			jsonTextPackage: jsonTextPackage, stringsPackage: stringsPackage,
			fmtPackage: fmtPackage, externalJSONTo: externalJSONTo,
			adjacentJSONTo: adjacentJSONTo,
		},
	)
	if err != nil {
		return nil, err
	}
	if firstEnumEdit >= 0 {
		edits[firstEnumEdit].text = enumJSONHelpers(
			jsonV2Package, jsonTextPackage, externalJSONTo, adjacentJSONTo,
			jsonUse.external, jsonUse.adjacent,
		) + edits[firstEnumEdit].text
	}
	edits, successLocations := lowerSuccessReturnCommas(files, file, tree, edits)
	edits, failureLocations := lowerFailureReturnCommas(files, file, tree, edits)
	edits, propagations, comprehensions, err := lowerCheckedExtensions(
		files, file, tree, name, data, defaultMarker, used, edits,
	)
	if err != nil {
		return nil, err
	}
	if jsonUse.enum {
		offset := file.Offset(tree.Name.Stop)
		imports := fmt.Sprintf(
			"\nimport %s \"encoding/json\"\n"+
				"import %s \"encoding/json/v2\"\n"+
				"import %s \"encoding/json/jsontext\"\n"+
				"import %s \"fmt\"\n",
			jsonPackage, jsonV2Package, jsonTextPackage, fmtPackage,
		)
		if jsonUse.foldedAdjacent {
			imports += fmt.Sprintf("import %s \"strings\"\n", stringsPackage)
		}
		edits = append(edits, edit{start: offset, end: offset, text: imports})
	}
	input := applyEdits(string(data), edits)
	mode := parser.ParseComments | parser.AllErrors | parser.SkipObjectResolution
	goFile, err := parser.ParseFile(files, name, input, mode)
	if err != nil {
		return nil, err
	}
	successReturns := projectedSuccessReturns(files, goFile, successLocations)
	if len(successReturns) != len(successLocations) {
		return nil, fmt.Errorf("parse %s: cannot project successful return", name)
	}
	failureReturns := projectedFailureReturns(files, goFile, failureLocations)
	if len(failureReturns) != len(failureLocations) {
		return nil, fmt.Errorf("parse %s: cannot project failure return", name)
	}
	nonNil := make(map[token.Pos]bool)
	ast.Inspect(goFile, func(node ast.Node) bool {
		pointer, ok := node.(*ast.StarExpr)
		if !ok {
			return true
		}
		position := files.Position(pointer.Star)
		if nonNilLocations[[2]int{position.Line, position.Column}] {
			nonNil[pointer.Star] = true
		}
		return true
	})
	result := &source{
		JSONPackage: jsonPackage, JSONV2Package: jsonV2Package,
		JSONTextPackage: jsonTextPackage, StringsPackage: stringsPackage,
		FmtPackage: fmtPackage, ExternalJSONTo: externalJSONTo,
		AdjacentJSONTo: adjacentJSONTo,
		Name:           name, Data: append([]byte(nil), data...), Tree: tree, File: goFile,
		Models: models, DefaultMarker: defaultMarker, Propagations: propagations,
		Comprehensions: comprehensions,
		NonNil:         nonNil,
		SuccessReturns: successReturns,
		FailureReturns: failureReturns,
		GeneratedHelpers: map[string]bool{
			externalJSONTo: jsonUse.external,
			adjacentJSONTo: jsonUse.adjacent,
		},
		Lowered: editsNeedOutput(edits),
	}
	return result, nil
}

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
		return freshIdentifier("__tgo_"+node.Name.Name+"_external_json_to", used),
			freshIdentifier("__tgo_"+node.Name.Name+"_adjacent_json_to", used)
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
		}
		return item, "type " + item.Name + " struct {\n" +
			fieldDecls(sourceName, item.Fields) + "}\n", enumJSONUse{}, nil
	}
	return nil, "", enumJSONUse{}, nil
}

func lowerCheckedExtensions(
	files *token.FileSet,
	file *token.File,
	tree *syntax.File,
	name string,
	data []byte,
	defaultMarker string,
	used map[string]bool,
	edits []edit,
) (
	[]edit,
	map[string]propagationSource,
	map[string]comprehensionSource,
	error,
) {
	edits, err := lowerExhaustiveClauses(files, file, tree, data, edits)
	if err != nil {
		return nil, nil, nil, err
	}
	edits, propagations, comprehensions, err := lowerSourceExtensions(
		files, file, tree, name, data, defaultMarker, used, edits,
	)
	return edits, propagations, comprehensions, err
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
		for _, name := range []string{
			declaration.Name + item.Name.Name,
			declaration.Name + "Tag" + item.Name.Name,
		} {
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
		for _, field := range item.Fields {
			if len(field.Field.Names) == 0 &&
				embeddedFieldName(field.Field.Type) == declaration.Name {
				return fmt.Errorf("enum payload field %s conflicts with its constructor method",
					declaration.Name)
			}
			for _, name := range field.Field.Names {
				if name.Name == declaration.Name {
					return fmt.Errorf("enum payload field %s conflicts with its constructor method",
						declaration.Name)
				}
			}
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

// eraseNonNilTypes makes the Go spelling used inside generated model declarations.
func eraseNonNilTypes(
	files *token.FileSet,
	file *token.File,
	tree *syntax.File,
	data []byte,
) ([]byte, map[[2]int]bool) {
	erased := append([]byte(nil), data...)
	locations := make(map[[2]int]bool)
	for _, extension := range syntax.Extensions(tree) {
		node, ok := syntax.NonNilPointerTypeOf(extension)
		if !ok {
			continue
		}
		erased[file.Offset(node.Percent)] = '*'
		position := files.Position(node.Percent)
		locations[[2]int{position.Line, position.Column}] = true
	}
	return erased, locations
}

// lowerSourceExtensions builds Go edits and propagation metadata for one source file.
func lowerSourceExtensions(
	files *token.FileSet,
	file *token.File,
	tree *syntax.File,
	name string,
	data []byte,
	defaultMarker string,
	used map[string]bool,
	edits []edit,
) ([]edit, map[string]propagationSource, map[string]comprehensionSource, error) {
	propagations := make(map[string]propagationSource)
	comprehensionNodes := []*syntax.ComprehensionExpression(nil)
	for _, extension := range syntax.Extensions(tree) {
		if node, ok := syntax.ComprehensionExpressionOf(extension); ok {
			comprehensionNodes = append(comprehensionNodes, node)
			continue
		}
		if node, ok := syntax.NonNilPointerTypeOf(extension); ok {
			start := file.Offset(node.Percent)
			if index := coveringEdit(edits, start); index >= 0 {
				edits[index].projectionOnly = false
			} else {
				edits = append(edits, edit{start: start, end: start + 1, text: "*"})
			}
			continue
		}
		if node, ok := syntax.DefaultExpressionOf(extension); ok {
			edits = append(edits, edit{
				start: file.Offset(node.Start), end: file.Offset(node.Stop),
				text: defaultMarker + ": true",
			})
			continue
		}
		node, ok := syntax.PropagationExpressionOf(extension)
		if !ok {
			continue
		}
		marker := freshIdentifier("__tgo_propagate", used)
		metadata, err := propagationMetadata(files, node)
		if err != nil {
			return nil, nil, nil, err
		}
		start := file.Offset(syntax.ExpressionPosition(node.Expression))
		bang := file.Offset(node.Bang)
		operatorEnd := bang + 1
		endPosition := files.Position(node.Bang)
		if metadata.Transparent {
			operatorEnd = file.Offset(node.SecondBang) + 1
			endPosition = files.Position(node.SecondBang)
		}
		startPosition := files.Position(syntax.ExpressionPosition(node.Expression))
		opening := marker + "(" + inlineLineDirective(
			name, startPosition.Line, startPosition.Column,
		)
		closing := ")" + inlineLineDirective(
			name, endPosition.Line, endPosition.Column+1,
		)
		edits = append(edits,
			edit{start: start, end: start, text: opening},
			edit{start: bang, end: operatorEnd, text: closing},
		)
		propagations[marker] = metadata
	}
	comprehensions := make(map[string]comprehensionSource)
	for _, node := range comprehensionNodes {
		marker := freshIdentifier("__tgo_comprehension", used)
		result := freshIdentifier("__tgo_result", used)
		replacement := comprehensionProjection(
			files, file, node, data, edits, marker, result,
		)
		start := file.Offset(node.Start)
		end := file.Offset(node.Stop)
		kept := edits[:0]
		for _, change := range edits {
			if start <= change.start && change.end <= end {
				continue
			}
			kept = append(kept, change)
		}
		edits = kept
		edits = append(edits, edit{start: start, end: end, text: replacement})
		comprehensions[marker] = comprehensionSource{
			Position: node.Start,
			Map:      node.Result.Key != nil,
		}
	}
	return edits, propagations, comprehensions, nil
}

func propagationMetadata(
	files *token.FileSet,
	node *syntax.PropagationExpression,
) (propagationSource, error) {
	transparent := node.SecondBang != token.NoPos
	name, static := syntax.StaticCallName(node.Call.Callee)
	if !transparent && !static {
		return propagationSource{}, fmt.Errorf(
			"%s: error propagation needs a short static call name",
			files.Position(node.Bang),
		)
	}
	return propagationSource{
		Bang: node.Bang, Name: name, Transparent: transparent,
	}, nil
}

// comprehensionProjection makes valid Go for type checking before direct lowering.
func comprehensionProjection(
	files *token.FileSet,
	file *token.File,
	node *syntax.ComprehensionExpression,
	data []byte,
	edits []edit,
	marker string,
	result string,
) string {
	var output strings.Builder
	typeText := editedSourceText(files, file, data, node.Type, edits)
	mapResult := node.Result.Key != nil
	output.WriteString(marker)
	output.WriteString("(func() ")
	output.WriteString(typeText)
	output.WriteString(" {\n")
	output.WriteString(result)
	if mapResult {
		output.WriteString(" := make(")
		output.WriteString(typeText)
		output.WriteString(")\n")
	} else {
		output.WriteString(" := make(")
		output.WriteString(typeText)
		output.WriteString(", 0)\n")
	}
	for _, clause := range node.Clauses {
		if rangeClause, ok := syntax.ComprehensionRangeClauseOf(&clause); ok {
			output.WriteString("for ")
			for index, binding := range rangeClause.Bindings {
				if index > 0 {
					output.WriteString(", ")
				}
				output.WriteString(binding.Name)
			}
			output.WriteString(" := range ")
			output.WriteString(editedSourceText(
				files, file, data, rangeClause.Source, edits,
			))
			output.WriteString(" {\n")
			continue
		}
		filter, _ := syntax.ComprehensionFilterClauseOf(&clause)
		output.WriteString("if ")
		output.WriteString(editedSourceText(
			files, file, data, filter.Condition, edits,
		))
		output.WriteString(" {\n")
	}
	if mapResult {
		output.WriteString(result)
		output.WriteString("[")
		output.WriteString(editedSourceText(
			files, file, data, node.Result.Key, edits,
		))
		output.WriteString("] = ")
		output.WriteString(editedSourceText(
			files, file, data, node.Result.Value, edits,
		))
		output.WriteByte('\n')
	} else {
		output.WriteString(result)
		output.WriteString(" = append(")
		output.WriteString(result)
		output.WriteString(", ")
		output.WriteString(editedSourceText(
			files, file, data, node.Result.Value, edits,
		))
		output.WriteString(")\n")
	}
	for range node.Clauses {
		output.WriteString("}\n")
	}
	output.WriteString("return ")
	output.WriteString(result)
	output.WriteString("\n})")
	position := files.Position(node.Stop)
	output.WriteString(inlineLineDirective(
		position.Filename, position.Line, position.Column,
	))
	return output.String()
}

func editedSourceText(
	files *token.FileSet,
	file *token.File,
	data []byte,
	expression *syntax.Expression,
	edits []edit,
) string {
	start := file.Offset(syntax.ExpressionPosition(expression))
	end := file.Offset(syntax.ExpressionEnd(expression))
	for {
		extended := false
		for _, change := range edits {
			if change.start == end && change.end > end {
				end = change.end
				extended = true
			}
		}
		if !extended {
			break
		}
	}
	local := make([]edit, 0)
	for _, change := range edits {
		if change.start < start || change.end > end {
			continue
		}
		local = append(local, edit{
			start: change.start - start,
			end:   change.end - start,
			text:  change.text,
		})
	}
	text := applyEdits(string(data[start:end]), local)
	position := files.Position(syntax.ExpressionPosition(expression))
	return inlineLineDirective(position.Filename, position.Line, position.Column) + text
}

func coveringEdit(edits []edit, offset int) int {
	for index, change := range edits {
		if change.start <= offset && offset < change.end {
			return index
		}
	}
	return -1
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

func generatedSource(name string, startLine int, endLine int, code string) string {
	return fmt.Sprintf("//line %s:%d:1\n%s\n//line %s:%d:1\n", name, startLine, code, name, endLine)
}

func lineDirective(name string, line int, column int) string {
	return fmt.Sprintf("//line %s:%d:%d\n", name, line, column)
}

func inlineLineDirective(name string, line int, column int) string {
	return fmt.Sprintf("/*line %s:%d:%d*/", name, line, column)
}

func freshIdentifier(base string, used map[string]bool) string {
	name := base
	for suffix := 1; used[name]; suffix++ {
		name = base + "_" + strconv.Itoa(suffix)
	}
	used[name] = true
	return name
}
