package frontend

import (
	"go/ast"
	"go/token"
	"reflect"
)

// finish maps syntax positions to the input and attaches parsed nodes to feature records.
// It replaces postfix comment markers with calls that the checker can identify.
func (result *Parsed) finish(fset *token.FileSet, file *ast.File, prepared, original *token.File,
	offsets []int) *Parsed {
	mapPositions(file, prepared, original, offsets, make(map[ast.Node]bool))
	result.File = file
	result.recordComprehensions(file)
	for expression, operation := range bindPostfixCalls(fset, file, result.Propagations) {
		result.Propagations[operation.Name] = operation
		replacePostfixOperand(file, original, expression, operation)
	}
	return result
}

// mapPositions changes every syntax position to its input position.
// The visited map prevents a shared node from being changed twice.
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
