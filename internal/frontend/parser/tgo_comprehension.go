package parser

import (
	"go/ast"
	"go/token"
)

// parseTGoLiteralValue reads ordinary literal elements or one output followed by clauses.
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
	return p.comprehensionCall(typ, lbrace, rbrace, elements[0], clauses)
}

// parseTGoElements distinguishes a comprehension after its first output.
// Ordinary literals keep Go's comma rules and diagnostics.
func (p *parser) parseTGoElements() ([]ast.Expr, []ast.Stmt) {
	var elements []ast.Expr
	for p.tok != token.RBRACE && p.tok != token.EOF {
		elements = append(elements, p.parseElement())
		if len(elements) == 1 {
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
		}
		if !p.atComma("composite literal", token.RBRACE) {
			break
		}
		p.next()
	}
	return elements, nil
}

// parseComprehensionClauses reads ranges and filters in their written order.
func (p *parser) parseComprehensionClauses() []ast.Stmt {
	var clauses []ast.Stmt
	for p.tok == token.FOR || p.tok == token.IF {
		if p.tok == token.FOR {
			clauses = append(clauses, p.parseComprehensionRange())
		} else {
			position := p.expect(token.IF)
			clauses = append(clauses, &ast.IfStmt{If: position, Cond: p.parseRhs()})
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
func (p *parser) parseComprehensionRange() ast.Stmt {
	position := p.expect(token.FOR)
	if p.tok == token.RANGE {
		rangePosition := p.expect(token.RANGE)
		return &ast.RangeStmt{For: position, Range: rangePosition, X: p.parseRhs()}
	}
	statement, isRange := p.parseSimpleStmt(rangeOk)
	if !isRange {
		p.error(position, "comprehension requires a range clause")
		return &ast.BadStmt{From: position, To: statement.End()}
	}
	assignment := statement.(*ast.AssignStmt)
	if len(assignment.Lhs) > 2 {
		p.errorExpected(assignment.Lhs[len(assignment.Lhs)-1].Pos(), "at most 2 expressions")
		return &ast.BadStmt{From: position, To: assignment.End()}
	}
	rangeExpression := assignment.Rhs[0].(*ast.UnaryExpr)
	result := &ast.RangeStmt{
		For: position, TokPos: assignment.TokPos, Tok: assignment.Tok,
		Range: rangeExpression.OpPos, X: rangeExpression.X,
	}
	if len(assignment.Lhs) > 0 {
		result.Key = assignment.Lhs[0]
	}
	if len(assignment.Lhs) == 2 {
		result.Value = assignment.Lhs[1]
	}
	return result
}

// comprehensionCall nests clauses around the output and records their exact nodes.
func (p *parser) comprehensionCall(typ ast.Expr, lbrace, rbrace token.Pos,
	element ast.Expr, clauses []ast.Stmt) *ast.CallExpr {
	arguments := []ast.Expr{element}
	if entry, ok := element.(*ast.KeyValueExpr); ok {
		arguments = []ast.Expr{entry.Key, entry.Value}
	}
	output := syntaxCall(element.Pos(), p.sourceExpressionEnd(element)-1, arguments)
	statements := []ast.Stmt{&ast.ExprStmt{X: output}}
	var ranges []*ast.RangeStmt
	for index := len(clauses) - 1; index >= 0; index-- {
		body := &ast.BlockStmt{Lbrace: clauses[index].Pos(), List: statements, Rbrace: rbrace}
		switch clause := clauses[index].(type) {
		case *ast.RangeStmt:
			clause.Body = body
		case *ast.IfStmt:
			clause.Body = body
		}
		statements = []ast.Stmt{clauses[index]}
	}
	for _, statement := range clauses {
		if clause, ok := statement.(*ast.RangeStmt); ok {
			ranges = append(ranges, clause)
		}
	}
	body := &ast.BlockStmt{Lbrace: lbrace, List: statements, Rbrace: rbrace}
	call := syntaxCall(typ.Pos(), rbrace, []ast.Expr{typ, syntaxFunction(lbrace, body)})
	if p.forms != nil {
		p.forms.Comprehensions = append(p.forms.Comprehensions,
			Comprehension{Call: call, Output: output, Position: typ.Pos(), Ranges: ranges})
	}
	return call
}
