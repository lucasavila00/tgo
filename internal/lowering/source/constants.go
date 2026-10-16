package source

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
)

// constantParents attaches checked initializer copies to their source declarations.
// Copies share source offsets, so expression sites keep the original syntax.
func (s *Inventory) constantParents(root ast.Node) {
	ast.Inspect(root, func(node ast.Node) bool {
		specification, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for _, name := range specification.Names {
			object, ok := s.Checked.TypesInfo.Defs[name].(*types.Const)
			if !ok {
				continue
			}
			initializer := s.Checked.TypesInfo.ConstInitializers[object]
			s.constantCopy(initializer.Type, specification)
			s.constantCopy(initializer.Value, specification)
		}
		return true
	})
}

// constantCopy records parent links for one inherited expression without changing source nodes.
func (s *Inventory) constantCopy(expression ast.Expr, parent ast.Node) {
	if expression == nil || s.Parents[expression] != nil {
		return
	}
	s.ConstantCopies = append(s.ConstantCopies, expression)
	stack := []ast.Node{parent}
	ast.Inspect(expression, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		s.Parents[node] = stack[len(stack)-1]
		stack = append(stack, node)
		return true
	})
	s.constantParents(expression)
}

// CallName returns the same source context for an operation and its inherited constant copies.
func (s *Inventory) CallName(operation *ast.PropagateExpr) string {
	return s.callNames[operation.OpPos]
}
