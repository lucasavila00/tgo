package shared

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"

	"github.com/lucasavila00/tgo/internal/frontend"
)

// RegisterComprehension connects generated ranges and output calls in this function.
func (r *Context) RegisterComprehension(record *frontend.Comprehension, destination *CollectionDestination) {
	if r.functionState.comprehensionRanges == nil {
		r.functionState.comprehensionRanges = make(map[*ast.RangeStmt]bool)
		r.functionState.collectionOutputs = make(map[string]*CollectionDestination)
	}
	for _, loop := range record.Ranges {
		r.functionState.comprehensionRanges[loop] = true
	}
	r.functionState.collectionOutputs[record.Output] = destination
}

// CollectionOutput finds an output destination in the current function.
func (r *Context) CollectionOutput(name string) *CollectionDestination {
	return r.functionState.collectionOutputs[name]
}

// CollectionDestination records storage for a collection output in one function.
type CollectionDestination struct {
	Index  *ast.Ident
	Grow   *ast.Ident
	Result *ast.Ident
	Type   types.Type
}

// BuiltinHidden checks source declarations before emitting a builtin name.
func (r *Context) BuiltinHidden(name string) bool {
	for _, object := range r.Input.Inventory.Checked.TypesInfo.Defs {
		if object != nil && object.Name() == name {
			return true
		}
	}
	for _, object := range r.Input.Inventory.Checked.TypesInfo.Implicits {
		if object != nil && object.Name() == name {
			return true
		}
	}
	return false
}

// ComprehensionBindings gives generator variables fresh names. Checked object
// identity keeps their output uses and closures separate from outer bindings.
func (r *Context) ComprehensionBindings(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncLit); ok && r.RealFunction(function) {
			return false
		}
		if loop, ok := node.(*ast.RangeStmt); ok {
			for _, expression := range []ast.Expr{loop.Key, loop.Value} {
				if id, ok := expression.(*ast.Ident); ok && id.Name != "_" {
					r.freshBinding(id)
				}
			}
		}
		return true
	})
}
