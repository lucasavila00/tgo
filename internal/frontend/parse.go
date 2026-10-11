package frontend

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"reflect"
)

// Parsed contains Go syntax and records for the added TGo expressions.
type Parsed struct {
	File           *ast.File
	Comprehensions map[string]*Comprehension
	Conditionals   map[string]token.Position
	Propagations   map[string]Marker
}

type mappedSource struct {
	input   string
	text    string
	offsets []int
}

func originalSource(input []byte) mappedSource {
	s := mappedSource{input: string(input), text: string(input), offsets: make([]int, len(input)+1)}
	for i := range s.offsets {
		s.offsets[i] = i
	}
	return s
}

func (s *mappedSource) replace(start, end int, text string, offsets []int) {
	s.text = s.text[:start] + text + s.text[end:]
	updatedOffsets := append([]int(nil), s.offsets[:start]...)
	updatedOffsets = append(updatedOffsets, offsets...)
	updatedOffsets = append(updatedOffsets, s.offsets[end:]...)
	s.offsets = updatedOffsets
}

func insertedOffsets(count, offset int) []int {
	result := make([]int, count)
	for i := range result {
		result[i] = offset
	}
	return result
}

// ParseFile parses added expressions and maps syntax positions to the input.
// Conditional bodies remain statements in the original enclosing function.
func ParseFile(fset *token.FileSet, filename, sourceName string, input []byte, used map[string]bool) (*Parsed, error) {
	result := &Parsed{Comprehensions: make(map[string]*Comprehension), Conditionals: make(map[string]token.Position), Propagations: make(map[string]Marker)}
	original := fset.AddFile(filename, -1, len(input))
	if sourceName != filename {
		original.AddLineColumnInfo(0, sourceName, 1, 1)
	}
	var scan scanner.Scanner
	scan.Init(original, input, nil, scanner.ScanComments)
	for {
		_, tok, literal := scan.Scan()
		if tok == token.IDENT {
			used[literal] = true
		}
		if tok == token.EOF {
			break
		}
	}
	source := prepareSource(sourceName, input, used, result.Propagations)
	for {
		for {
			conditional := source.prepareConditionals(filename, original, used, result.Conditionals)
			changed, err := source.prepareComprehension(original, used, result.Comprehensions)
			if err != nil {
				return nil, err
			}
			if !changed && !conditional {
				break
			}
		}
		temporary := token.NewFileSet()
		file, err := parser.ParseFile(temporary, filename, source.text, parser.ParseComments|parser.AllErrors|parser.SkipObjectResolution)
		if err == nil {
			return result.finish(fset, file, temporary.File(file.Pos()), original, source.offsets), nil
		}
		errors, ok := err.(scanner.ErrorList)
		if !ok {
			return nil, err
		}
		start := conditionalStart(file, temporary, source.text)
		if start < 0 {
			for _, diagnostic := range errors {
				diagnostic.Pos = original.Position(original.Pos(source.offsets[diagnostic.Pos.Offset]))
			}
			return nil, errors
		}
		if err := source.wrapConditional(filename, original, start, used, result.Conditionals); err != nil {
			return nil, err
		}
	}
}

func prepareSource(sourceName string, input []byte, used map[string]bool, propagations map[string]Marker) mappedSource {
	source := originalSource(input)
	// Postfix comments let Go parse the surrounding expression. Build the same
	// replacement as MarkPostfix while retaining every original byte offset.
	MarkPostfix(sourceName, input, used, propagations)
	// Find edits in source order; MarkPostfix reserves names in scan order.
	var edits []struct {
		start, end int
		name       string
	}
	for name, op := range propagations {
		end := op.Position.Offset + 1
		if end < len(input) && input[end] == '!' {
			end++
		}
		edits = append(edits, struct {
			start, end int
			name       string
		}{op.Position.Offset, end, name})
	}
	for len(edits) > 0 {
		index := 0
		for i := range edits {
			if edits[i].start > edits[index].start {
				index = i
			}
		}
		edit := edits[index]
		text := "/*" + edit.name + "*/"
		source.replace(edit.start, edit.end, text, insertedOffsets(len(text), edit.start))
		edits = append(edits[:index], edits[index+1:]...)
	}
	return source
}

func (result *Parsed) finish(fset *token.FileSet, file *ast.File, prepared, original *token.File, offsets []int) *Parsed {
	mapPositions(file, prepared, original, offsets, make(map[ast.Node]bool))
	result.File = file
	ast.Inspect(file, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok {
				if record := result.Comprehensions[id.Name]; record != nil {
					record.Type = call.Args[0]
					record.Body = call.Args[1].(*ast.FuncLit).Body
					record.Ranges = comprehensionRanges(record.Body)
				}
			}
		}
		return true
	})
	for expression, operation := range MarkerExpressions(fset, file, result.Propagations) {
		result.Propagations[operation.Name] = operation
		WrapMarker(file, expression, operation)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok {
				if op, found := result.Propagations[id.Name]; found {
					width := 1
					if op.Wrap {
						width++
					}
					call.Rparen = original.Pos(op.Position.Offset + width - 1)
				}
			}
		}
		return true
	})
	return result
}

func mapPositions(node ast.Node, prepared, original *token.File, offsets []int, visited map[ast.Node]bool) {
	if node == nil || reflect.ValueOf(node).IsNil() {
		return
	}
	if visited[node] {
		return
	}
	visited[node] = true
	value := reflect.ValueOf(node).Elem()
	positionType := reflect.TypeOf(token.Pos(0))
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Type() == positionType {
			pos := token.Pos(field.Int())
			if pos.IsValid() {
				field.SetInt(int64(original.Pos(offsets[prepared.Offset(pos)])))
			}
		} else if child, ok := field.Interface().(ast.Node); ok {
			mapPositions(child, prepared, original, offsets, visited)
		} else if field.Kind() == reflect.Slice {
			for j := 0; j < field.Len(); j++ {
				if child, ok := field.Index(j).Interface().(ast.Node); ok {
					mapPositions(child, prepared, original, offsets, visited)
				}
			}
		}
	}
}
