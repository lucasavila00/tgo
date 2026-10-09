package tgolint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestNilParallelAssignmentKeepsValueIdentity(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		unsafe bool
	}{
		{
			name: "move saved proof",
			body: "checked := other != nil\n" +
				"value, other = other, value\n" +
				"if checked { need(value) }",
		},
		{
			name: "keep proof on copied value",
			body: "other = value\nalias = value\n" +
				"checked := value != nil\nvalue = nil\n" +
				"if checked { need(other); need(alias) }",
		},
		{
			name: "swap different values",
			body: "value = &Item{}\nother = nil\n" +
				"value, other = other, value\nneed(value)",
			unsafe: true,
		},
		{
			name: "swap keeps non-nil value",
			body: "value = nil\nother = &Item{}\n" +
				"value, other = other, value\nneed(value)",
		},
	}
	for _, test := range tests {
		diagnostics, err := runNilAssignmentAnalysis(test.body)
		if err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if got := len(diagnostics) != 0; got != test.unsafe {
			t.Fatalf(
				"%s: got %d diagnostics; want unsafe=%t",
				test.name, len(diagnostics), test.unsafe,
			)
		}
	}
}

func runNilAssignmentAnalysis(body string) ([]analysis.Diagnostic, error) {
	source := fmt.Sprintf(`package sample
type Item struct{}
func need(value *Item) {}
func subject(value, other, alias *Item) {
%s
}
`, body)
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "sample.go", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Scopes:     make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	pkg, err := new(types.Config).Check("sample", set, []*ast.File{file}, info)
	if err != nil {
		return nil, err
	}
	diagnostics := []analysis.Diagnostic(nil)
	pass := &analysis.Pass{
		Fset: set, Files: []*ast.File{file}, Pkg: pkg, TypesInfo: info,
		Report: func(diagnostic analysis.Diagnostic) {
			diagnostics = append(diagnostics, diagnostic)
		},
		ImportObjectFact: func(types.Object, analysis.Fact) bool { return false },
	}
	environment := newNilEnvironment(pass, []*ast.File{file}, info, pkg, nil)
	environment.collectNilContracts()
	need, _ := pkg.Scope().Lookup("need").(*types.Func)
	environment.contracts[need] = nilContract{"p0": true}
	environment.checkNilFiles()
	return diagnostics, nil
}
