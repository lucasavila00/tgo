package parser

import (
	"go/ast"
	"go/token"
)

// Forms records added syntax by its exact parsed nodes and source positions.
// The frontend assigns helper names after the complete file parses.
type Forms struct {
	Postfix        []Postfix
	Conditionals   []Conditional
	Comprehensions []Comprehension
	ShortReturns   []ShortReturn
}

// Postfix identifies the call operand and the source span of ! or !!.
type Postfix struct {
	Operand    ast.Expr
	Start, End token.Pos
	Wrap       bool
}

// Conditional identifies the syntax container for a value-producing if or switch.
type Conditional struct {
	Call     *ast.CallExpr
	Position token.Pos
}

// Comprehension identifies the collection, its output, and its ordered ranges.
type Comprehension struct {
	Call, Output *ast.CallExpr
	Position     token.Pos
	Ranges       []*ast.RangeStmt
}

// ShortReturn identifies omitted results and the source position of .. .
type ShortReturn struct {
	Statement   *ast.ReturnStmt
	Call        *ast.CallExpr
	Position    token.Pos
	ZeroSuccess bool
}
