// Package source records evaluation sites and parent links in checked syntax.
// It keeps source facts separate from the state used to emit Go statements.
package source

import (
	"fmt"
	"github.com/lucasavila00/tgo/internal/checker"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
	"path/filepath"
)

// Site identifies an expression in the original source. Offsets are bytes.
type Site struct {
	File         string `json:"file"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Kind         string `json:"kind"`
	Reason       string `json:"reason,omitempty"`
	ErrorReturn  bool   `json:"error_return"`
	ReturnReason string `json:"return_reason,omitempty"`
}

// Inventory records sites and their original syntax. Callers must keep this syntax unchanged.
type Inventory struct {
	Checked     *checker.Package
	Sites       []Site
	Expressions map[Site]ast.Expr
	Parents     map[ast.Node]ast.Node
}

// New records source facts from one checked package.
func New(checked *checker.Package, dir string) *Inventory {
	inventory := &Inventory{Checked: checked, Expressions: make(map[Site]ast.Expr), Parents: make(map[ast.Node]ast.Node)}
	for _, file := range checked.Syntax {
		inventory.inventory(file, dir)
	}
	return inventory
}

func (s *Inventory) inventory(file *ast.File, dir string) {
	var stack []ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) != 0 {
			s.Parents[n] = stack[len(stack)-1]
		}
		if expr, ok := n.(ast.Expr); ok {
			// Use physical file names and offsets for expression sites.
			start := s.Checked.Fset.PositionFor(expr.Pos(), false)
			end := s.Checked.Fset.PositionFor(expr.End(), false)
			name, err := filepath.Rel(dir, start.Filename)
			if err != nil {
				name = start.Filename
			}
			site := Site{File: name, Start: start.Offset, End: end.Offset, Kind: fmt.Sprintf("%T", expr)}
			tv, hasType := s.Checked.TypesInfo.Types[expr]
			switch {
			case hasType && tv.IsBuiltin():
				site.Reason = "builtin name"
			case hasType && tv.IsType():
				site.Reason = "type syntax"
			case !hasType:
				site.Reason = "name or syntax without a value"
			}
			if assertion, ok := expr.(*ast.TypeAssertExpr); ok && assertion.Type == nil {
				site.Reason = "type switch assertion syntax"
			}
			var fn ast.Node
			for i := len(stack) - 1; i >= 0; i-- {
				switch parent := stack[i].(type) {
				case *ast.FuncDecl:
					fn = parent
				case *ast.FuncLit:
					call, _ := s.Parents[parent].(*ast.CallExpr)
					if s.Checked.Conditionals[call] == nil && s.Checked.Comprehensions[call] == nil {
						fn = parent
					}
				case *ast.GenDecl:
					if parent.Tok == token.CONST {
						site.Reason = "constant declaration"
					}
				case *ast.ArrayType:
					if contains(parent.Len, expr) {
						site.Reason = "array length"
					}
				case *ast.RangeStmt:
					if contains(parent.X, expr) && RangeElided(s.Checked, parent) {
						site.Reason = "range expression is not evaluated"
					}
				case *ast.GoStmt:
					if parent.Call == expr {
						site.Reason = "call executes in another goroutine"
					}
				case *ast.DeferStmt:
					if parent.Call == expr {
						site.Reason = "call executes when the function returns"
					}
				}
				if fn != nil {
					break
				}
			}
			if fn == nil && site.Reason == "" {
				site.Reason = "package scope"
			}
			if fn != nil {
				var typ *ast.FuncType
				switch f := fn.(type) {
				case *ast.FuncDecl:
					typ = f.Type
				case *ast.FuncLit:
					typ = f.Type
				}
				sig, _ := s.Checked.TypesInfo.TypeOf(typ).(*types.Signature)
				if sig == nil {
					if f, ok := fn.(*ast.FuncDecl); ok {
						sig, _ = s.Checked.TypesInfo.Defs[f.Name].Type().(*types.Signature)
					}
					if f, ok := fn.(*ast.FuncLit); ok {
						sig, _ = s.Checked.TypesInfo.TypeOf(f).(*types.Signature)
					}
				}
				if sig != nil && sig.Results().Len() == 1 {
					site.ErrorReturn = types.Identical(sig.Results().At(0).Type(), types.Universe.Lookup("error").Type())
				}
				if !site.ErrorReturn {
					site.ReturnReason = "function does not return only error"
				}
			} else {
				site.ReturnReason = "no enclosing function"
			}
			s.Sites = append(s.Sites, site)
			s.Expressions[site] = expr
		}
		stack = append(stack, n)
		return true
	})
}

// RangeElided reports whether Go omits evaluation of a range expression.
func RangeElided(checked *checker.Package, statement *ast.RangeStmt) bool {
	if statement.Value != nil {
		return false
	}
	if tv := checked.TypesInfo.Types[statement.X]; tv.Value != nil {
		return true
	}
	typ := checked.TypesInfo.TypeOf(statement.X).Underlying()
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem().Underlying()
	}
	if _, ok := typ.(*types.Array); !ok {
		return false
	}
	constantLength := true
	ast.Inspect(statement.X, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok && checked.TypesInfo.Types[call].Value == nil && !checked.TypesInfo.Types[call.Fun].IsType() {
			constantLength = false
		}
		if unary, ok := n.(*ast.UnaryExpr); ok && unary.Op == token.ARROW {
			constantLength = false
		}
		return true
	})
	return constantLength
}

func contains(parent, child ast.Node) bool {
	return parent != nil && child != nil && parent.Pos() <= child.Pos() && child.End() <= parent.End()
}
