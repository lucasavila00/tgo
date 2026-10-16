package types

import (
	"go/token"
	"testing"

	"github.com/lucasavila00/tgo/internal/ast"
)

// TestConstantCopyDropsParserBindings avoids legacy binding cycles while keeping source positions.
func TestConstantCopyDropsParserBindings(t *testing.T) {
	identifier := &ast.Ident{NamePos: 10, Name: "iota"}
	expression := &ast.BinaryExpr{X: identifier, OpPos: 15, Op: token.ADD,
		Y: &ast.BasicLit{ValuePos: 16, Kind: token.INT, Value: "1"}}
	identifier.Obj = &ast.Object{Decl: expression} //nolint:staticcheck // Test removal of a deprecated parser binding.
	copied := copyConstExpression(expression).(*ast.BinaryExpr)
	if copied == expression || copied.X == identifier || copied.Y == expression.Y {
		t.Fatal("copy retained a source node")
	}
	if copied.X.(*ast.Ident).Obj != nil {
		t.Fatal("copy retained the parser binding object")
	}
	if copied.Pos() != expression.Pos() || copied.End() != expression.End() {
		t.Fatal("copy changed source positions")
	}
}
