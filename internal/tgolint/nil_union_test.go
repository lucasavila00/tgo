package tgolint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"math/rand"
	"testing"
	"testing/quick"

	"golang.org/x/tools/go/analysis"
)

func TestNilTypeLatticeProperties(t *testing.T) {
	property := func(leftByte, middleByte, rightByte uint8) bool {
		left := nilTypeFromMembers(leftByte)
		middle := nilTypeFromMembers(middleByte)
		right := nilTypeFromMembers(rightByte)
		return equalNilType(unionNilTypes(left, middle), unionNilTypes(middle, left)) &&
			equalNilType(
				unionNilTypes(unionNilTypes(left, middle), right),
				unionNilTypes(left, unionNilTypes(middle, right)),
			) &&
			equalNilType(intersectNilTypes(left, middle), intersectNilTypes(middle, left)) &&
			equalNilType(
				intersectNilTypes(intersectNilTypes(left, middle), right),
				intersectNilTypes(left, intersectNilTypes(middle, right)),
			) &&
			equalNilType(unionNilTypes(left, left), left) &&
			equalNilType(intersectNilTypes(left, left), left)
	}
	configuration := &quick.Config{
		MaxCount: 1_000,
		Rand:     rand.New(rand.NewSource(1)), //nolint:gosec // Tests need stable data.
	}
	if err := quick.Check(property, configuration); err != nil {
		t.Fatal(err)
	}
}

func TestDeclaredNilTypeSeparatesStringAndOptionalString(t *testing.T) {
	stringType := types.Typ[types.String]
	if !isNonNilType(declaredNilType(stringType)) {
		t.Fatal("string must exclude nil")
	}
	optionalString := types.NewPointer(stringType)
	if !isOptionalNilType(declaredNilType(optionalString)) {
		t.Fatal("*string must contain string and nil")
	}
	if !isNonNilType(intersectNilTypes(optionalNilType(), nonNilType())) {
		t.Fatal("a nil check must remove nil from *string")
	}
}

func TestNilBooleanReachability(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		unsafe bool
	}{
		{
			name: "contradictory branch",
			body: "if value != nil && value == nil { need(value) }",
		},
		{
			name: "saved false guard",
			body: "value = nil\nchecked := value != nil\n" +
				"value = nil\nif checked { need(value) }",
		},
		{
			name:   "unsafe branch",
			body:   "if value == nil { need(value) }",
			unsafe: true,
		},
		{
			name: "invalidated true guard",
			body: "value = &Item{}\nchecked := value != nil\n" +
				"value = nil\nif checked { need(value) }",
			unsafe: true,
		},
	}
	for _, test := range tests {
		diagnostics, err := runNilAnalysis("value *Item", test.body)
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
