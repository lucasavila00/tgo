package compiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestFunctionNamesIncludeClosureUses(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "names.go", `package sample
func use() {
	_ = func() any { return captured }
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	function := file.Decls[0].(*ast.FuncDecl)
	signature := types.NewSignatureType(
		nil, nil, nil, types.NewTuple(), types.NewTuple(), false,
	)
	if !functionNames(function.Body, signature)["captured"] {
		t.Fatal("closure use was not reserved")
	}
}
