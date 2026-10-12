package lowering

import (
	"go/ast"
	"reflect"
)

// renameBindings copies paths to moved loop bindings. Object identity keeps
// outer variables and shadowed names unchanged, including closure references.
func (r *rewrite) renameBindings(node ast.Node) ast.Node {
	if node == nil || len(r.file.bindings) == 0 {
		return node
	}
	if id, ok := node.(*ast.Ident); ok {
		object := r.input.source.inventory.Checked.TypesInfo.Uses[id]
		if object == nil {
			object = r.input.source.inventory.Checked.TypesInfo.Defs[id]
		}
		if name := r.file.bindings[object]; name != "" {
			renamed := *id
			renamed.Name = name
			return &renamed
		}
		return node
	}
	original := reflect.ValueOf(node).Elem()
	rewritten := reflect.New(original.Type())
	rewritten.Elem().Set(original)
	changed := rewriteChildFields(original, rewritten.Elem(), r.renameBindings)
	if !changed {
		return node
	}
	return rewritten.Interface().(ast.Node)
}
