package lowering

import (
	"go/ast"
	"reflect"
)

// renameBindings copies paths to moved loop bindings. Object identity keeps
// outer variables and shadowed names unchanged, including closure references.
func (r *rewrite) renameBindings(node ast.Node) ast.Node {
	if node == nil || len(r.bindings) == 0 {
		return node
	}
	if id, ok := node.(*ast.Ident); ok {
		object := r.source.checked.TypesInfo.Uses[id]
		if object == nil {
			object = r.source.checked.TypesInfo.Defs[id]
		}
		if name := r.bindings[object]; name != "" {
			copy := *id
			copy.Name = name
			return &copy
		}
		return node
	}
	original := reflect.ValueOf(node).Elem()
	copy := reflect.New(original.Type())
	copy.Elem().Set(original)
	changed := false
	for i := 0; i < original.NumField(); i++ {
		field, output := original.Field(i), copy.Elem().Field(i)
		if child, ok := field.Interface().(ast.Node); ok && !reflect.ValueOf(child).IsNil() {
			if rewritten := r.renameBindings(child); rewritten != child {
				output.Set(reflect.ValueOf(rewritten))
				changed = true
			}
		} else if field.Kind() == reflect.Slice {
			for j := 0; j < field.Len(); j++ {
				child, ok := field.Index(j).Interface().(ast.Node)
				if !ok || reflect.ValueOf(child).IsNil() {
					continue
				}
				if rewritten := r.renameBindings(child); rewritten != child {
					if output.Pointer() == field.Pointer() {
						output.Set(reflect.MakeSlice(field.Type(), field.Len(), field.Len()))
						reflect.Copy(output, field)
					}
					output.Index(j).Set(reflect.ValueOf(rewritten))
					changed = true
				}
			}
		}
	}
	if !changed {
		return node
	}
	return copy.Interface().(ast.Node)
}
