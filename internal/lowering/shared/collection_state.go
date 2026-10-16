package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
)

// CollectionDestination records storage for a collection output in one function.
type CollectionDestination struct {
	Index  *ast.Ident
	Grow   *ast.Ident
	Result *ast.Ident
	Type   types.Type
}

// BuiltinHidden checks source declarations before emitting a builtin name.
func (r *Context) BuiltinHidden(name string) bool {
	for _, object := range r.input.Inventory.Checked.TypesInfo.Defs {
		if object != nil && object.Name() == name {
			return true
		}
	}
	for _, object := range r.input.Inventory.Checked.TypesInfo.Implicits {
		if object != nil && object.Name() == name {
			return true
		}
	}
	return false
}

// ComprehensionBindings gives generator variables fresh names. Checked object
// identity keeps their output uses and closures separate from outer bindings.
func (r *Context) ComprehensionBindings(expression *ast.ComprehensionExpr) {
	for _, clause := range expression.Clauses {
		switch clause := clause.(type) {
		case *ast.RangeClause:
			for _, binding := range []ast.Expr{clause.Key, clause.Value} {
				if identifier, ok := binding.(*ast.Ident); ok && identifier.Name != "_" {
					r.freshBinding(identifier)
				}
			}
			r.nestedComprehensionBindings(clause.X)
		case *ast.FilterClause:
			r.nestedComprehensionBindings(clause.Cond)
		}
	}
	for _, output := range expression.Output {
		r.nestedComprehensionBindings(output)
	}
}

// nestedComprehensionBindings records nested generators before their output storage is allocated.
func (r *Context) nestedComprehensionBindings(node ast.Node) {
	ast.Inspect(node, func(child ast.Node) bool {
		switch child := child.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ComprehensionExpr:
			r.ComprehensionBindings(child)
			return false
		}
		return true
	})
}

// ComprehensionOperand adds output parentheses and saves a range source when requested.
func (r *Context) ComprehensionOperand(expression ast.Expr, capture bool) Result {
	evaluation := r.Expression(expression)
	value := evaluation.Values[0]
	parenthesized := &ast.ParenExpr{Lparen: expression.Pos(), X: value.Expression, Rparen: expression.End() - 1}
	value.Expression = parenthesized
	value.Temporary = false
	value.Original = expression
	r.recordValue(value)
	evaluation.Values[0] = value
	if capture {
		checked := r.input.Inventory.Checked.TypesInfo.Types[expression]
		if checked.Value == nil && !checked.IsNil() {
			saved := r.Save(value)
			evaluation.Statements = append(evaluation.Statements, saved.Statements...)
			evaluation.Values = saved.Values
		}
	}
	return evaluation
}

// ComprehensionRange emits a range for one source clause with its prepared body.
func (r *Context) ComprehensionRange(clause *ast.RangeClause, body *ast.BlockStmt) []ast.Stmt {
	exit := r.prepareBreakExit(clause, clause.X)
	evaluation := r.RangeOperand(clause.X, clause.Value)
	operand := &ast.ParenExpr{Lparen: clause.X.Pos(), X: evaluation.Expressions()[0], Rparen: clause.X.End() - 1}
	output := &ast.RangeStmt{For: clause.For, Key: clause.Key, Value: clause.Value, TokPos: clause.TokPos,
		Tok: clause.Tok, Range: clause.Range, X: operand, Body: body}
	return r.ComprehensionStatements(r.withBreakExit(clause, output, append(evaluation.Statements, output), exit))
}

// ComprehensionStatements keeps generated locals out of paths entered by a source goto.
func (r *Context) ComprehensionStatements(statements []ast.Stmt) []ast.Stmt {
	if r.functionState.hasGoto && len(statements) > 1 {
		return []ast.Stmt{&ast.BlockStmt{List: statements}}
	}
	return statements
}
