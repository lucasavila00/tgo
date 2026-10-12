package lowering

import (
	"fmt"
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"go/ast"

	"github.com/lucasavila00/tgo/internal/lowering/source"
)

// InsertAfter returns Go files with the callback's statements after the selected expression.
// The callback receives saved results; the original syntax tree stays unchanged.
func (s *Source) InsertAfter(site Site, after After) (map[string][]byte, error) {
	target, ok := s.inventory.Expressions[source.Site(site)]
	if !ok {
		return nil, fmt.Errorf("expression site does not belong to this source")
	}
	if site.Reason != "" {
		return nil, fmt.Errorf("expression has no current-function evaluation: %s", site.Reason)
	}
	r := newRewrite(s)
	r.Input.Target = target
	r.Input.After = shared.After(after)
	output, err := r.Emit()
	if err != nil {
		return nil, err
	}
	if !r.Request.Inserted {
		return nil, fmt.Errorf("expression site was not lowered: %s:%d", site.File, site.Start)
	}
	return output, nil
}

// LowerPropagations replaces checked TGo operations with ordinary Go.
// It returns formatted file contents by absolute path without writing files.
// It reports an error if any accepted operation was not replaced.
func (s *Source) LowerPropagations(operations map[string]Propagation) (map[string][]byte, error) {
	checked := s.inventory.Checked
	r := newRewrite(s)
	r.Input.Propagations = make(map[ast.Expr]shared.Propagation)
	r.Input.Comprehensions = checked.Comprehensions
	r.Input.Conditionals = checked.Conditionals
	r.Input.ShortReturns = checked.ShortReturns
	r.Request.LoweredReturns = make(map[*ast.ReturnStmt]bool)
	r.Request.Lowered = make(map[ast.Expr]bool)
	for site, expression := range s.inventory.Expressions {
		call, ok := expression.(*ast.CallExpr)
		if !ok {
			continue
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok {
			continue
		}
		operation, ok := operations[id.Name]
		if !ok {
			continue
		}
		if site.Reason != "" {
			return nil, fmt.Errorf("propagation has no current-function evaluation: %s", site.Reason)
		}
		r.Input.Propagations[call] = shared.Propagation(operation)
	}
	output, err := r.Emit()
	if err != nil {
		return nil, err
	}
	if err := r.CheckLoweredOperations(); err != nil {
		return nil, err
	}
	return output, nil
}
