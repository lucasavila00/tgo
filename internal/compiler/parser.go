package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"tgo/syntax"
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
	syntax.Inspect(tree, func(node syntax.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok {
			used[identifier.Name] = true
		}
		return true
	})
	if used["__tgo_runtime"] {
		return nil, fmt.Errorf(
			"%s: generated validation import name __tgo_runtime is reserved",
			name,
		)
	}
	matchMarker := freshIdentifier("__tgo_match", used)
	defaultMarker := freshIdentifier("__tgo_defaults", used)
	runtimeAlias := "__tgo_runtime"
	file := files.File(tree.Package)
	edits := []edit(nil)
	models := []*model(nil)
	for _, declaration := range tree.Decls {
		var item *model
		var replacement string
		switch node := declaration.(type) {
		case *syntax.EnumDecl:
			item = enumModel(files, data, node)
			replacement = enumGo(name, item, runtimeAlias)
		case *syntax.StructDecl:
			item = structModel(files, data, node)
			replacement = "type " + item.Name + " struct {\n" +
				fieldDecls(name, item.Fields) + "}\n"
			replacement += structValidationGo(item, runtimeAlias)
		case *syntax.CheckedDecl:
			item = checkedModel(files, data, node)
			replacement = checkedGo(name, item, runtimeAlias)
		}
		if item == nil {
			continue
		}
		start := file.Offset(declaration.Pos())
		end := declarationEditEnd(data, file.Offset(declaration.End()))
		startPosition := files.Position(declaration.Pos())
		endPosition := files.Position(declaration.End())
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
		switch node := extension.(type) {
		case *syntax.MatchStmt:
			matchStart := file.Offset(node.Match)
			matchEnd := matchStart + len("match")
			brace := file.Offset(node.Lbrace)
			edits = append(edits,
				edit{start: matchStart, end: matchEnd, text: "switch " + matchMarker + "("},
				edit{start: brace, end: brace, text: ") "},
			)
		case *syntax.DefaultMarker:
			edits = append(edits, edit{
				start: file.Offset(node.Pos()),
				end:   file.Offset(node.End()),
				text:  defaultMarker + ": true",
			})
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
		RuntimeAlias:  runtimeAlias,
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

func enumModel(files *token.FileSet, data []byte, declaration *syntax.EnumDecl) *model {
	position := files.Position(declaration.Name.Pos())
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

func structModel(files *token.FileSet, data []byte, declaration *syntax.StructDecl) *model {
	position := files.Position(declaration.Name.Pos())
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

func checkedModel(files *token.FileSet, data []byte, declaration *syntax.CheckedDecl) *model {
	position := files.Position(declaration.Name.Pos())
	basePosition := files.Position(declaration.Base.Pos())
	predicatePosition := files.Position(declaration.Predicate.Pos())
	return &model{
		Name:            declaration.Name.Name,
		Enum:            false,
		Line:            position.Line,
		Column:          position.Column,
		Base:            sourceText(files, data, declaration.Base),
		BaseLine:        basePosition.Line,
		BaseColumn:      basePosition.Column,
		Predicate:       sourceText(files, data, declaration.Predicate),
		PredicateLine:   predicatePosition.Line,
		PredicateColumn: predicatePosition.Column,
		Variants:        nil,
		Fields:          nil,
	}
}

func modelFields(files *token.FileSet, data []byte, declarations []*syntax.FieldDecl) []field {
	result := []field(nil)
	for _, declaration := range declarations {
		typeText := sourceText(files, data, declaration.Field.Type)
		typePosition := files.Position(declaration.Field.Type.Pos())
		defaultText := ""
		defaultLine := 0
		defaultColumn := 0
		if declaration.Default != nil {
			defaultText = sourceText(files, data, declaration.Default)
			defaultPosition := files.Position(declaration.Default.Pos())
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

func sourceText(files *token.FileSet, data []byte, node ast.Node) string {
	file := files.File(node.Pos())
	return string(data[file.Offset(node.Pos()):file.Offset(node.End())])
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
