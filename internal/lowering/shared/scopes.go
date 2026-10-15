package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/token"
)

// ScopeOf returns the checked evaluation scope, including comprehension bindings.
func (r *Context) ScopeOf(expression ast.Expr) *types.Scope {
	original := r.ValueOf(expression).Original
	if scope := r.input.Inventory.Checked.TypesInfo.ExprScopes[original]; scope != nil {
		return scope
	}
	return r.input.Inventory.Checked.Types.Scope().Innermost(original.Pos())
}

// scopeAt uses the source evaluation scope for operands and the requested scope for function headers.
func (r *Context) scopeAt(expression ast.Expr, position token.Pos) *types.Scope {
	if expression.Pos() == position {
		return r.ScopeOf(expression)
	}
	return r.input.Inventory.Checked.Types.Scope().Innermost(position)
}

// VisibleBinding finds a source name that remains visible after moved bindings get fresh names.
func (r *Context) VisibleBinding(expression ast.Expr, name string) types.Object {
	original := r.ValueOf(expression).Original
	return r.lookupBinding(r.ScopeOf(expression), name, original.Pos())
}

// lookupBinding skips source bindings whose output names are already different.
func (r *Context) lookupBinding(scope *types.Scope, name string, position token.Pos) types.Object {
	for scope != nil {
		owner, object := scope.LookupParent(name, position)
		emittedName := r.file.bindings[object]
		if object == nil || emittedName == "" || emittedName == object.Name() {
			return object
		}
		scope = owner.Parent()
	}
	return nil
}
