package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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
	reserveImportNames(tree, used)
	defaultMarker := freshIdentifier("tgoDefaults", used)
	imports := planEnumImports(tree, used)
	externalJSONTo, adjacentJSONTo := enumJSONHelperNames(tree, used)
	file := files.File(tree.Package)
	erasedData, nonNilLocations := eraseNonNilTypes(files, file, tree, data)
	models, edits, jsonUse, firstEnumEdit, err := sourceModels(
		files, file, tree, name, data, erasedData,
		sourceModelConfig{
			jsonPackage: imports.json.name, jsonV2Package: imports.jsonV2.name,
			jsonTextPackage: imports.jsonText.name,
			stringsPackage:  imports.strings.name,
			fmtPackage:      imports.fmt.name, externalJSONTo: externalJSONTo,
			adjacentJSONTo: adjacentJSONTo,
		},
	)
	if err != nil {
		return nil, err
	}
	if firstEnumEdit >= 0 {
		edits[firstEnumEdit].text = enumJSONHelpers(
			imports.jsonV2.name, imports.jsonText.name,
			externalJSONTo, adjacentJSONTo,
			jsonUse.external, jsonUse.adjacent,
		) + edits[firstEnumEdit].text
	}
	edits, successLocations := lowerSuccessReturnCommas(files, file, tree, edits)
	edits, failureLocations := lowerFailureReturnCommas(files, file, tree, edits)
	edits, propagations, comprehensions, exhaustive, err := lowerCheckedExtensions(
		files, file, tree, name, data, defaultMarker, used, edits,
	)
	if err != nil {
		return nil, err
	}
	edits = imports.addEdits(edits, file, tree.Name.Stop, jsonUse)
	input := applyEdits(string(data), edits)
	mode := parser.ParseComments | parser.AllErrors | parser.SkipObjectResolution
	goFile, err := parser.ParseFile(files, name, input, mode)
	if err != nil {
		return nil, err
	}
	lowerExhaustiveReceiverEvaluations(goFile, exhaustive)
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
		JSONPackage: imports.json.name, JSONV2Package: imports.jsonV2.name,
		JSONTextPackage: imports.jsonText.name,
		StringsPackage:  imports.strings.name,
		FmtPackage:      imports.fmt.name, ExternalJSONTo: externalJSONTo,
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
	map[string]string,
	error,
) {
	edits, exhaustive, err := lowerExhaustiveClauses(
		files, file, tree, data, used, edits,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	edits, propagations, comprehensions, err := lowerSourceExtensions(
		files, file, tree, name, data, defaultMarker, used, edits,
	)
	return edits, propagations, comprehensions, exhaustive, err
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
		marker := freshIdentifier("tgoPropagate", used)
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
		marker := freshIdentifier("tgoComprehension", used)
		result := freshIdentifier("tgoResult", used)
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
			Position: node.Start, Result: result, Map: node.Result.Key != nil,
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

func generatedSource(name string, startLine int, endLine int, code string) string {
	return fmt.Sprintf("//line %s:%d:1\n%s\n//line %s:%d:1\n", name, startLine, code, name, endLine)
}

func lineDirective(name string, line int, column int) string {
	return fmt.Sprintf("//line %s:%d:%d\n", name, line, column)
}

func inlineLineDirective(name string, line int, column int) string {
	return fmt.Sprintf("/*line %s:%d:%d*/", name, line, column)
}
