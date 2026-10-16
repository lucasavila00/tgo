package parser

import (
	"go/token"

	"github.com/lucasavila00/tgo/internal/ast"
)

// parseEnumType reads variant declarations without adding generated Go declarations.
func (p *parser) parseEnumType() *ast.EnumType {
	declaration := &ast.EnumType{Enum: p.pos}
	p.next()
	declaration.Lbrace = p.expect(token.LBRACE)
	for p.tok != token.RBRACE && p.tok != token.EOF {
		variant := &ast.EnumVariant{Name: p.parseIdent()}
		if p.tok == token.STRUCT {
			variant.Type = p.parseStructType()
		}
		declaration.Variants = append(declaration.Variants, variant)
		p.expectSemi()
	}
	declaration.Rbrace = p.expect(token.RBRACE)
	return declaration
}

// startsExhaustiveClause recognizes the label while keeping ordinary identifiers valid.
func (p *parser) startsExhaustiveClause() bool {
	if !p.tgo || p.tok != token.IDENT || p.lit != "exhaustive" {
		return false
	}
	next, _ := p.peekHeaderToken(0)
	return next.Token == token.COLON
}

// parseCaseStatements stops at a following case or exhaustive label.
func (p *parser) parseCaseStatements() (statements []ast.Stmt) {
	for p.tok != token.CASE && p.tok != token.DEFAULT && p.tok != token.RBRACE && p.tok != token.EOF {
		if p.startsExhaustiveClause() {
			break
		}
		statements = append(statements, p.parseStmt())
	}
	return statements
}
