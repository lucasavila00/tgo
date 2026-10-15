package parser

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"
)

// parseTGoLiteralValue reads ordinary literal elements or output operands followed by clauses.
// Clause statements retain source positions and share the enclosing source function.
func (p *parser) parseTGoLiteralValue(typ ast.Expr) ast.Expr {
	lbrace := p.expect(token.LBRACE)
	p.exprLev++
	elements, clauses := p.parseTGoElements()
	p.exprLev--
	rbrace := p.expectClosing(token.RBRACE, "composite literal")
	if len(clauses) == 0 {
		return &ast.CompositeLit{Type: typ, Lbrace: lbrace, Elts: elements, Rbrace: rbrace}
	}
	if typ == nil {
		p.error(lbrace, "comprehension requires a collection type")
		return &ast.BadExpr{From: lbrace, To: rbrace + 1}
	}
	return &ast.ComprehensionExpr{Type: typ, Lbrace: lbrace, Rbrace: rbrace, Output: elements, Clauses: clauses}
}

// parseTGoElements records all output operands before comprehension clauses.
// Ordinary literals keep Go's comma rules and diagnostics.
func (p *parser) parseTGoElements() ([]ast.Expr, []ast.ComprehensionClause) {
	var elements []ast.Expr
	for p.tok != token.RBRACE && p.tok != token.EOF {
		if p.tok == token.FOR {
			return elements, p.parseComprehensionClauses()
		}
		elements = append(elements, p.parseElement())
		if p.tok == token.FOR {
			return elements, p.parseComprehensionClauses()
		}
		if p.tok == token.SEMICOLON && p.lit == "\n" {
			newline := p.pos
			p.next()
			if p.tok == token.FOR {
				return elements, p.parseComprehensionClauses()
			}
			p.error(newline, "missing ',' before newline in composite literal")
			continue
		}
		if !p.atComma("composite literal", token.RBRACE) {
			break
		}
		p.next()
	}
	return elements, nil
}

// parseComprehensionClauses reads ranges and filters in their written order.
func (p *parser) parseComprehensionClauses() []ast.ComprehensionClause {
	var clauses []ast.ComprehensionClause
	for p.tok == token.FOR || p.tok == token.IF {
		if p.tok == token.FOR {
			clauses = append(clauses, p.parseComprehensionRange())
		} else {
			position := p.expect(token.IF)
			condition := p.parseRhs()
			clauses = append(clauses, &ast.FilterClause{If: position, Cond: condition, EndPos: condition.End()})
		}
		for p.tok == token.SEMICOLON {
			p.next()
		}
	}
	if p.tok == token.COMMA {
		p.next()
	}
	return clauses
}

// parseComprehensionRange uses Go's range header grammar without a loop body.
func (p *parser) parseComprehensionRange() ast.ComprehensionClause {
	position := p.expect(token.FOR)
	if p.tok == token.RANGE {
		rangePosition := p.expect(token.RANGE)
		source := p.parseRhs()
		return &ast.RangeClause{For: position, Range: rangePosition, X: source, EndPos: source.End()}
	}
	statement, isRange := p.parseSimpleStmt(rangeOk)
	if !isRange {
		p.error(position, "comprehension requires a range clause")
		return &ast.RangeClause{For: position, X: &ast.BadExpr{From: position, To: statement.End()}, EndPos: statement.End()}
	}
	assignment := statement.(*ast.AssignStmt)
	if len(assignment.Lhs) > 2 {
		p.errorExpected(assignment.Lhs[len(assignment.Lhs)-1].Pos(), "at most 2 expressions")
		return &ast.RangeClause{
			For: position, X: &ast.BadExpr{From: position, To: assignment.End()}, EndPos: assignment.End(),
		}
	}
	rangeExpression := assignment.Rhs[0].(*ast.UnaryExpr)
	result := &ast.RangeClause{
		For: position, TokPos: assignment.TokPos, Tok: assignment.Tok,
		Range: rangeExpression.OpPos, X: rangeExpression.X, EndPos: rangeExpression.X.End(),
	}
	if len(assignment.Lhs) > 0 {
		result.Key = assignment.Lhs[0]
	}
	if len(assignment.Lhs) == 2 {
		result.Value = assignment.Lhs[1]
	}
	return result
}
