package shared

import (
	"go/ast"
	"reflect"
)

// rewriteChildFields copies changed AST children without changing input slices.
func rewriteChildFields(original, output reflect.Value, visit func(ast.Node) ast.Node) bool {
	changed := false
	for i := 0; i < original.NumField(); i++ {
		field, destination := original.Field(i), output.Field(i)
		if child, ok := field.Interface().(ast.Node); ok {
			if reflect.ValueOf(child).IsNil() {
				continue
			}
			rewritten := visit(child)
			if rewritten != child {
				destination.Set(reflect.ValueOf(rewritten))
				changed = true
			}
		} else if field.Kind() == reflect.Slice {
			if rewriteChildSlice(field, destination, visit) {
				changed = true
			}
		}
	}
	return changed
}

// rewriteChildSlice visits AST children and copies the destination slice only when a child changes.
func rewriteChildSlice(original, output reflect.Value, visit func(ast.Node) ast.Node) bool {
	changed := false
	for i := 0; i < original.Len(); i++ {
		child, ok := original.Index(i).Interface().(ast.Node)
		if !ok || reflect.ValueOf(child).IsNil() {
			continue
		}
		rewritten := visit(child)
		if rewritten == child {
			continue
		}
		if !changed {
			output.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Len()))
			reflect.Copy(output, original)
		}
		output.Index(i).Set(reflect.ValueOf(rewritten))
		changed = true
	}
	return changed
}
