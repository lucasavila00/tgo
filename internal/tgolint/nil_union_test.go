package tgolint

import (
	"fmt"
	"go/types"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"testing/quick"

	"tgo/internal/sourcefacts"
	"tgo/pkg/syntax"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
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
		diagnostics, err := runNilAnalysis(t, "value *Item", test.body)
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

func runNilAnalysis(
	t *testing.T,
	parameters string,
	body string,
) ([]analysis.Diagnostic, error) {
	t.Helper()
	source := fmt.Sprintf(`package sample
type Item struct{}
func need(value *Item) {}
func subject(%s) {
%s
}
`, parameters, body)
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"), []byte("module sample\n"), 0o600,
	); err != nil {
		return nil, err
	}
	filename := filepath.Join(directory, "sample.go")
	if err := os.WriteFile(filename, []byte(source), 0o600); err != nil {
		return nil, err
	}
	loaded, err := packages.Load(&packages.Config{
		Dir: directory,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
	}, ".")
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(loaded) != 0 || len(loaded) != 1 {
		return nil, fmt.Errorf("load sample package")
	}
	loadedPackage := loaded[0]
	file, err := syntax.ParseGoFile(
		loadedPackage.Fset, filename, []byte(source), syntax.ParseComments|syntax.AllErrors,
	)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, fmt.Errorf("load sample package syntax")
	}
	fileSet := loadedPackage.Fset
	if fileSet == nil {
		return nil, fmt.Errorf("load sample package file set")
	}
	typeInfo := loadedPackage.TypesInfo
	if typeInfo == nil {
		return nil, fmt.Errorf("load sample package type facts")
	}
	packageTypes := loadedPackage.Types
	if packageTypes == nil {
		return nil, fmt.Errorf("load sample package facts")
	}
	diagnostics := []analysis.Diagnostic(nil)
	pass := &analysis.Pass{
		Fset: fileSet, Pkg: packageTypes,
		Report: func(diagnostic analysis.Diagnostic) {
			diagnostics = append(diagnostics, diagnostic)
		},
		ImportObjectFact: func(types.Object, analysis.Fact) bool { return false },
	}
	facts := sourcefacts.New(file, typeInfo, fileSet)
	environment := newNilEnvironment(
		pass, singleNilEnvironmentFile(file), facts, packageTypes, nil,
	)
	environment.collectNilContracts()
	need, _ := packageTypes.Scope().Lookup("need").(*types.Func)
	environment.contracts[need] = nilContract{"p0": true}
	environment.checkNilFiles()
	return diagnostics, nil
}
