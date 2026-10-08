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
	matchMarker := freshIdentifier("__tgo_match", used)
	defaultMarker := freshIdentifier("__tgo_defaults", used)
	file := files.File(tree.Package)
	erasedData, nonNilLocations := eraseNonNilTypes(files, file, tree, data)
	edits := []edit(nil)
	models := []*model(nil)
	for _, declaration := range tree.Declarations {
		var item *model
		var replacement string
		if node, ok := syntax.EnumDeclarationOf(declaration); ok {
			item = enumModel(files, erasedData, node)
			replacement = enumGo(name, item)
		} else if node, ok := syntax.StructDeclarationOf(declaration); ok {
			item = structModel(files, erasedData, node)
			replacement = "type " + item.Name + " struct {\n" +
				fieldDecls(name, item.Fields) + "}\n"
		} else if node, ok := syntax.CheckedDeclarationOf(declaration); ok {
			item = checkedModel(files, erasedData, node)
			replacement = checkedGo(name, item)
		}
		if item == nil {
			continue
		}
		startPositionValue := syntax.DeclarationPosition(declaration)
		endPositionValue := syntax.DeclarationEnd(declaration)
		start := file.Offset(startPositionValue)
		end := declarationEditEnd(data, file.Offset(endPositionValue))
		startPosition := files.Position(startPositionValue)
		endPosition := files.Position(endPositionValue)
		edits = append(edits, edit{
			start: start,
			end:   end,
			text: generatedSource(
				name,
				startPosition.Line,
				endPosition.Line,
				replacement,
			),
		})
		models = append(models, item)
	}
	edits, propagations, comprehensions, err := lowerSourceExtensions(
		files, file, tree, name, data, matchMarker, defaultMarker, used, edits,
	)
	if err != nil {
		return nil, err
	}
	input := applyEdits(string(data), edits)
	mode := parser.ParseComments | parser.AllErrors | parser.SkipObjectResolution
	goFile, err := parser.ParseFile(files, name, input, mode)
	if err != nil {
		return nil, err
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
	return &source{
		Name:           name,
		Data:           append([]byte(nil), data...),
		Tree:           tree,
		File:           goFile,
		Models:         models,
		MatchMarker:    matchMarker,
		DefaultMarker:  defaultMarker,
		Propagations:   propagations,
		Comprehensions: comprehensions,
		NonNil:         nonNil,
	}, nil
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
	matchMarker string,
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
			if !coveredByEdit(edits, start) {
				edits = append(edits, edit{start: start, end: start + 1, text: "*"})
			}
			continue
		}
		if node, ok := syntax.MatchStatementOf(extension); ok {
			matchStart := file.Offset(node.Match)
			brace := file.Offset(node.Lbrace)
			edits = append(edits,
				edit{
					start: matchStart, end: matchStart + len("match"),
					text: "switch " + matchMarker + "(",
				},
				edit{start: brace, end: brace, text: ") "},
			)
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
		callName, ok := syntax.StaticCallName(node.Call.Callee)
		if !ok {
			return nil, nil, nil, fmt.Errorf(
				"%s: error propagation needs a short static call name",
				files.Position(node.Bang),
			)
		}
		start := file.Offset(syntax.ExpressionPosition(node.Expression))
		bang := file.Offset(node.Bang)
		startPosition := files.Position(syntax.ExpressionPosition(node.Expression))
		bangPosition := files.Position(node.Bang)
		opening := marker + "(" + inlineLineDirective(
			name, startPosition.Line, startPosition.Column,
		)
		closing := ")" + inlineLineDirective(
			name, bangPosition.Line, bangPosition.Column+1,
		)
		edits = append(edits,
			edit{start: start, end: start, text: opening},
			edit{start: bang, end: bang + 1, text: closing},
		)
		propagations[marker] = propagationSource{Bang: node.Bang, Name: callName}
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

func coveredByEdit(edits []edit, offset int) bool {
	for _, change := range edits {
		if change.start <= offset && offset < change.end {
			return true
		}
	}
	return false
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
		Name:            declaration.Name.Name,
		Enum:            true,
		Line:            position.Line,
		Column:          position.Column,
		Base:            "",
		BaseLine:        0,
		BaseColumn:      0,
		Predicate:       "",
		PredicateLine:   0,
		PredicateColumn: 0,
		Variants:        nil,
		Fields:          nil,
	}
	for _, item := range declaration.Variants {
		result.Variants = append(result.Variants, variant{
			Name:   item.Name.Name,
			Fields: modelFields(files, data, item.Fields),
		})
	}
	return result
}

func structModel(files *token.FileSet, data []byte, declaration *syntax.StructDeclaration) *model {
	position := files.Position(declaration.Name.Start)
	return &model{
		Name:            declaration.Name.Name,
		Enum:            false,
		Line:            position.Line,
		Column:          position.Column,
		Base:            "",
		BaseLine:        0,
		BaseColumn:      0,
		Predicate:       "",
		PredicateLine:   0,
		PredicateColumn: 0,
		Variants:        nil,
		Fields:          modelFields(files, data, declaration.Fields),
	}
}

func checkedModel(
	files *token.FileSet,
	data []byte,
	declaration *syntax.CheckedDeclaration,
) *model {
	position := files.Position(declaration.Name.Start)
	basePosition := files.Position(syntax.ExpressionPosition(declaration.Base))
	predicatePosition := files.Position(syntax.ExpressionPosition(declaration.Predicate))
	return &model{
		Name:   declaration.Name.Name,
		Enum:   false,
		Line:   position.Line,
		Column: position.Column,
		Base: sourceText(
			files, data, syntax.ExpressionPosition(declaration.Base),
			syntax.ExpressionEnd(declaration.Base),
		),
		BaseLine:   basePosition.Line,
		BaseColumn: basePosition.Column,
		Predicate: sourceText(
			files, data, syntax.ExpressionPosition(declaration.Predicate),
			syntax.ExpressionEnd(declaration.Predicate),
		),
		PredicateLine:   predicatePosition.Line,
		PredicateColumn: predicatePosition.Column,
		Variants:        nil,
		Fields:          nil,
	}
}

func modelFields(files *token.FileSet, data []byte, declarations []*syntax.TGoField) []field {
	result := []field(nil)
	for _, declaration := range declarations {
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
			result = append(result, newField(
				"", typeText, tag, defaultText, typePosition, defaultLine, defaultColumn,
			))
			continue
		}
		for _, name := range declaration.Field.Names {
			result = append(result, newField(
				name.Name, typeText, tag, defaultText, typePosition, defaultLine, defaultColumn,
			))
		}
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
