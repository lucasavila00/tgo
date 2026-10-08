package compiler

import (
	"fmt"
	"go/parser"
	"go/token"
	"strconv"

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
	propagations := make(map[string]propagationSource)
	file := files.File(tree.Package)
	edits := []edit(nil)
	models := []*model(nil)
	for _, declaration := range tree.Declarations {
		var item *model
		var replacement string
		if node, ok := syntax.EnumDeclarationOf(declaration); ok {
			item = enumModel(files, data, node)
			replacement = enumGo(name, item)
		} else if node, ok := syntax.StructDeclarationOf(declaration); ok {
			item = structModel(files, data, node)
			replacement = "type " + item.Name + " struct {\n" +
				fieldDecls(name, item.Fields) + "}\n"
		} else if node, ok := syntax.CheckedDeclarationOf(declaration); ok {
			item = checkedModel(files, data, node)
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
	for _, extension := range syntax.Extensions(tree) {
		if node, ok := syntax.MatchStatementOf(extension); ok {
			matchStart := file.Offset(node.Match)
			matchEnd := matchStart + len("match")
			brace := file.Offset(node.Lbrace)
			edits = append(edits,
				edit{start: matchStart, end: matchEnd, text: "switch " + matchMarker + "("},
				edit{start: brace, end: brace, text: ") "},
			)
			continue
		}
		if node, ok := syntax.DefaultExpressionOf(extension); ok {
			edits = append(edits, edit{
				start: file.Offset(node.Start),
				end:   file.Offset(node.Stop),
				text:  defaultMarker + ": true",
			})
			continue
		}
		if node, ok := syntax.PropagationExpressionOf(extension); ok {
			marker := freshIdentifier("__tgo_propagate", used)
			callName, ok := syntax.StaticCallName(node.Call.Callee)
			if !ok {
				return nil, fmt.Errorf(
					"%s: error propagation needs a short static call name",
					files.Position(node.Bang),
				)
			}
			start := file.Offset(syntax.ExpressionPosition(node.Expression))
			bang := file.Offset(node.Bang)
			startPosition := files.Position(syntax.ExpressionPosition(node.Expression))
			bangPosition := files.Position(node.Bang)
			opening := marker + "(" + inlineLineDirective(
				name,
				startPosition.Line,
				startPosition.Column,
			)
			closing := ")" + inlineLineDirective(
				name,
				bangPosition.Line,
				bangPosition.Column+1,
			)
			edits = append(edits,
				edit{start: start, end: start, text: opening},
				edit{start: bang, end: bang + 1, text: closing},
			)
			propagations[marker] = propagationSource{Bang: node.Bang, Name: callName}
		}
	}
	input := applyEdits(string(data), edits)
	mode := parser.ParseComments | parser.AllErrors | parser.SkipObjectResolution
	goFile, err := parser.ParseFile(files, name, input, mode)
	if err != nil {
		return nil, err
	}
	return &source{
		Name:          name,
		Data:          append([]byte(nil), data...),
		File:          goFile,
		Models:        models,
		MatchMarker:   matchMarker,
		DefaultMarker: defaultMarker,
		Propagations:  propagations,
	}, nil
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
