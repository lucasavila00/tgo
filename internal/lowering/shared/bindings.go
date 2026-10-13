package shared

import (
	"go/ast"
	"reflect"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// renameBindings copies paths to moved loop bindings. Object identity keeps
// outer variables and shadowed names unchanged, including closure references.
func (r *Context) renameBindings(node ast.Node) ast.Node {
	if node == nil || len(r.file.bindings) == 0 {
		return node
	}
	if id, ok := node.(*ast.Ident); ok {
		object := r.input.Inventory.Checked.TypesInfo.Uses[id]
		if object == nil {
			object = r.input.Inventory.Checked.TypesInfo.Defs[id]
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

// freshBinding reserves one name per checked object, including closure uses.
func (r *Context) freshBinding(identifier *ast.Ident) {
	object := r.input.Inventory.Checked.TypesInfo.Defs[identifier]
	if object == nil {
		return
	}
	if r.file.bindings == nil {
		r.file.bindings = make(map[types.Object]string)
	}
	if r.file.bindings[object] == "" {
		r.file.bindings[object] = r.NameFor(identifier.Name).Name
	}
}

// BindingReference returns the emitted name for a checked source binding.
func (r *Context) BindingReference(object types.Object) *ast.Ident {
	return ast.NewIdent(r.file.bindings[object])
}
