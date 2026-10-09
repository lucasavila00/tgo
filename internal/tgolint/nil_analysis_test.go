package tgolint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// runNilAnalysis checks one valid Go function with a non-nil call target.
func runNilAnalysis(parameters string, body string) ([]analysis.Diagnostic, error) {
	source := fmt.Sprintf(`package sample
type Item struct{}
func need(value *Item) {}
func subject(%s) {
%s
}
`, parameters, body)
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

func formatDiagnostics(diagnostics []analysis.Diagnostic) string {
	messages := make([]string, len(diagnostics))
	for index, diagnostic := range diagnostics {
		messages[index] = diagnostic.Message
	}
	return strings.Join(messages, "\n")
}
