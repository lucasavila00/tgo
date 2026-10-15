// Package source records checked facts that lowering needs before statements move.
//
// Go syntax does not always evaluate an expression. An array length is constant;
// some range operands do not run; a deferred call runs when its function returns.
// The inventory records these limits with each expression's physical file and
// byte offsets. It also records the source function that owns a return.
//
// Parent links retain the original expression contexts. Checked scopes and jump
// targets keep their source identities after lowering moves statements.
// This input stays unchanged during emission. Each rewrite owns separate output
// state, so the same checked source can produce independent outputs.
package source

import (
	"fmt"
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker"
	"github.com/lucasavila00/tgo/internal/checker/types"
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
	inventory := &Inventory{Checked: checked, Expressions: make(map[Site]ast.Expr),
		Parents: make(map[ast.Node]ast.Node)}
	for _, file := range checked.Syntax {
		inventory.inventory(file, dir)
	}
	return inventory
}

// inventory visits one checked file and adds its source-parent links and expression records to the inventory.
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
			s.recordExpression(expr, stack, dir)
		}
		stack = append(stack, n)
		return true
	})
}

// RangeElided reports whether Go omits evaluation of a range expression.
func RangeElided(checked *checker.Package, statement *ast.RangeStmt) bool {
	return RangeOperandElided(checked, statement.X, statement.Value)
}

// RangeOperandElided reports whether range omits evaluation of its checked operand.
func RangeOperandElided(checked *checker.Package, expression, value ast.Expr) bool {
	if value != nil {
		return false
	}
	if tv := checked.TypesInfo.Types[expression]; tv.Value != nil {
		return true
	}
	typ := checked.TypesInfo.TypeOf(expression).Underlying()
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem().Underlying()
	}
	if _, ok := typ.(*types.Array); !ok {
		return false
	}
	constantLength := true
	ast.Inspect(expression, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.PropagateExpr, *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr:
			constantLength = false
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok && checked.TypesInfo.Types[call].Value == nil &&
			!checked.TypesInfo.Types[call.Fun].IsType() {
			constantLength = false
		}
		if unary, ok := n.(*ast.UnaryExpr); ok && unary.Op == token.ARROW {
			constantLength = false
		}
		return true
	})
	return constantLength
}

// contains reports whether the child's source position range lies inside the parent's range.
func contains(parent, child ast.Node) bool {
	return parent != nil && child != nil && parent.Pos() <= child.Pos() && child.End() <= parent.End()
}

// recordExpression records the expression location and whether an inserted statement or error return can run there.
func (s *Inventory) recordExpression(expr ast.Expr, stack []ast.Node, dir string) {
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
	enclosingFunction, reason := s.expressionContext(expr, stack)
	if reason != "" {
		site.Reason = reason
	}
	if enclosingFunction == nil && site.Reason == "" {
		site.Reason = "package scope"
	}
	site.ErrorReturn, site.ReturnReason = s.returnTarget(enclosingFunction)
	s.Sites = append(s.Sites, site)
	s.Expressions[site] = expr
}

// expressionContext finds the enclosing source function and evaluation limits from the original ancestor chain.
func (s *Inventory) expressionContext(expr ast.Expr, stack []ast.Node) (ast.Node, string) {
	var enclosingFunction ast.Node
	var reason string
	for i := len(stack) - 1; i >= 0; i-- {
		switch parent := stack[i].(type) {
		case *ast.FuncDecl:
			enclosingFunction = parent
		case *ast.FuncLit:
			enclosingFunction = parent
		default:
			if currentReason := s.evaluationReason(stack[i], expr); currentReason != "" {
				reason = currentReason
			}
		}
		if enclosingFunction != nil {
			return enclosingFunction, reason
		}
	}
	return enclosingFunction, reason
}

// returnTarget reports whether the enclosing function returns exactly one error value for an inserted error return.
func (s *Inventory) returnTarget(enclosingFunction ast.Node) (bool, string) {
	if enclosingFunction == nil {
		return false, "no enclosing function"
	}
	var functionType *ast.FuncType
	switch function := enclosingFunction.(type) {
	case *ast.FuncDecl:
		functionType = function.Type
	case *ast.FuncLit:
		functionType = function.Type
	}
	signature, _ := s.Checked.TypesInfo.TypeOf(functionType).(*types.Signature)
	if signature == nil {
		switch function := enclosingFunction.(type) {
		case *ast.FuncDecl:
			signature, _ = s.Checked.TypesInfo.Defs[function.Name].Type().(*types.Signature)
		case *ast.FuncLit:
			signature, _ = s.Checked.TypesInfo.TypeOf(function).(*types.Signature)
		}
	}
	if signature != nil && signature.Results().Len() == 1 &&
		types.Identical(signature.Results().At(0).Type(), types.Universe.Lookup("error").Type()) {
		return true, ""
	}
	return false, "function does not return only error"
}

// evaluationReason describes source contexts where the expression is constant, skipped, or executed later.
func (s *Inventory) evaluationReason(node ast.Node, expr ast.Expr) string {
	switch parent := node.(type) {
	case *ast.GenDecl:
		if parent.Tok == token.CONST {
			return "constant declaration"
		}
	case *ast.ArrayType:
		if contains(parent.Len, expr) {
			return "array length"
		}
	case *ast.RangeStmt:
		if contains(parent.X, expr) && RangeElided(s.Checked, parent) {
			return "range expression is not evaluated"
		}
	case *ast.RangeClause:
		if contains(parent.X, expr) && RangeOperandElided(s.Checked, parent.X, parent.Value) {
			return "range expression is not evaluated"
		}
	case *ast.GoStmt:
		if parent.Call == expr {
			return "call executes in another goroutine"
		}
	case *ast.DeferStmt:
		if parent.Call == expr {
			return "call executes when the function returns"
		}
	}
	return ""
}
