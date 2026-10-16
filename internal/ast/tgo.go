package ast

import "go/token"

// PropagateExpr propagates the final error from X; Wrap adds source call context.
type PropagateExpr struct {
	X            Expr
	OpPos, OpEnd token.Pos
	Wrap         bool
}

// IfExpr chooses one value from branches in the enclosing source function.
type IfExpr struct {
	If   token.Pos
	Init Stmt
	Cond Expr
	Body *BlockStmt
	Else Node // *BlockStmt or *IfExpr
}

// SwitchExpr produces a value from an ordinary switch or type switch.
type SwitchExpr struct {
	Switch Stmt // *SwitchStmt or *TypeSwitchStmt
}

// ComprehensionExpr builds a collection from output operands and ordered clauses.
type ComprehensionExpr struct {
	Type           Expr
	Lbrace, Rbrace token.Pos
	Output         []Expr
	Clauses        []ComprehensionClause
}

// ComprehensionClause is a range or filter in a collection comprehension.
type ComprehensionClause interface {
	Node
	comprehensionClause()
}

// RangeClause supplies bindings from one range source.
type RangeClause struct {
	For, Range token.Pos
	Key, Value Expr
	TokPos     token.Pos
	Tok        token.Token
	X          Expr
	EndPos     token.Pos
}

// FilterClause retains only the iterations for which Cond is true.
type FilterClause struct {
	If     token.Pos
	Cond   Expr
	EndPos token.Pos
}

// ShortReturnStmt supplies the error or success results and omits the other results.
type ShortReturnStmt struct {
	Return, Placeholder, PlaceholderEnd token.Pos
	Results                             []Expr
	ZeroSuccess                         bool
}

// Pos returns the start of the call operand.
func (expression *PropagateExpr) Pos() token.Pos { return expression.X.Pos() }

// End returns the byte position after the postfix operator.
func (expression *PropagateExpr) End() token.Pos { return expression.OpEnd }

// exprNode makes propagation part of the expression grammar.
func (*PropagateExpr) exprNode() {}

// Pos returns the if keyword's position.
func (expression *IfExpr) Pos() token.Pos { return expression.If }

// End returns the end of the final branch.
func (expression *IfExpr) End() token.Pos {
	if expression.Else != nil {
		return expression.Else.End()
	}
	return expression.Body.End()
}

// exprNode makes value-producing if part of the expression grammar.
func (*IfExpr) exprNode() {}

// Pos returns the switch keyword's position.
func (expression *SwitchExpr) Pos() token.Pos { return expression.Switch.Pos() }

// End returns the end of the switch body.
func (expression *SwitchExpr) End() token.Pos { return expression.Switch.End() }

// exprNode makes value-producing switch part of the expression grammar.
func (*SwitchExpr) exprNode() {}

// Pos returns the collection type's position.
func (expression *ComprehensionExpr) Pos() token.Pos { return expression.Type.Pos() }

// End returns the position after the collection's closing brace.
func (expression *ComprehensionExpr) End() token.Pos { return expression.Rbrace + 1 }

// exprNode makes comprehensions part of the expression grammar.
func (*ComprehensionExpr) exprNode() {}

// Pos returns the range clause's for keyword position.
func (clause *RangeClause) Pos() token.Pos { return clause.For }

// End returns the end of the complete range source.
func (clause *RangeClause) End() token.Pos { return clause.EndPos }

// comprehensionClause makes a range part of the comprehension grammar.
func (*RangeClause) comprehensionClause() {}

// Pos returns the filter's if keyword position.
func (clause *FilterClause) Pos() token.Pos { return clause.If }

// End returns the end of the complete filter condition.
func (clause *FilterClause) End() token.Pos { return clause.EndPos }

// comprehensionClause makes a filter part of the comprehension grammar.
func (*FilterClause) comprehensionClause() {}

// Pos returns the return keyword's position.
func (statement *ShortReturnStmt) Pos() token.Pos { return statement.Return }

// End returns the end of the last operand or placeholder.
func (statement *ShortReturnStmt) End() token.Pos {
	end := statement.PlaceholderEnd
	if len(statement.Results) != 0 && statement.Results[len(statement.Results)-1].End() > end {
		end = statement.Results[len(statement.Results)-1].End()
	}
	return end
}

// stmtNode makes shortened returns part of the statement grammar.
func (*ShortReturnStmt) stmtNode() {}
