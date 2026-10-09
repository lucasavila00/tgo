package compiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"tgo/internal/sourcefacts"
	"tgo/pkg/syntax"
)

func TestAnalysisOwnersCoverGeneratedPublicSurface(t *testing.T) {
	t.Parallel()
	packages, err := AnalyzeWorkspace("testdata/analysisworkspace")
	if err != nil {
		t.Fatal(err)
	}
	var analysis *AnalysisPackage
	for _, pkg := range packages {
		if pkg.Path == "example.test/analysis/dep" {
			analysis = pkg
		}
	}
	if analysis == nil {
		t.Fatal("dependency analysis is absent")
	}
	positions := modelOwnerPositions(t, analysis)
	objects := make(map[types.Object]token.Pos)
	scope := analysis.Package.Scope()
	for _, name := range []string{
		"ChoiceTag", "ChoiceTagOne", "ChoiceTagTwo", "ChoiceOne", "ChoiceTwo",
	} {
		objects[scope.Lookup(name)] = positions[name]
	}
	objects[scope.Lookup("NewPositive")] = positions["Positive"]
	for _, name := range []string{"Choice", "ChoiceOne", "ChoiceTwo", "Positive"} {
		named := namedObject(scope.Lookup(name))
		for index := 0; index < named.NumMethods(); index++ {
			method := named.Method(index)
			owner := positions[name]
			if name == "Choice" && strings.HasSuffix(method.Name(), "Payload") {
				owner = positions[strings.TrimSuffix(method.Name(), "Payload")]
			}
			objects[method] = owner
		}
	}
	if len(analysis.Owners) != len(objects) {
		t.Fatalf("owner count = %d, want %d", len(analysis.Owners), len(objects))
	}
	for object, want := range objects {
		if object == nil {
			t.Fatal("generated public object is absent")
		}
		if got := analysis.Owners[object]; got != want {
			t.Fatalf(
				"owner for %s = %s, want %s",
				object.Name(), analysis.Files.Position(got), analysis.Files.Position(want),
			)
		}
	}
}

func modelOwnerPositions(
	t *testing.T,
	analysis *AnalysisPackage,
) map[string]token.Pos {
	t.Helper()
	result := make(map[string]token.Pos)
	for _, source := range analysis.Sources {
		for _, declaration := range source.Syntax.Declarations {
			if enum, ok := syntax.EnumDeclarationOf(declaration); ok {
				result[enum.Name.Name] = enum.Name.Start
				result[enum.Name.Name+"Tag"] = enum.Name.Start
				for _, variant := range enum.Variants {
					result[variant.Name.Name] = variant.Name.Start
					result[enum.Name.Name+"Tag"+variant.Name.Name] = variant.Name.Start
					result[enum.Name.Name+variant.Name.Name] = variant.Name.Start
				}
			}
			if checked, ok := syntax.CheckedDeclarationOf(declaration); ok {
				result[checked.Name.Name] = checked.Name.Start
			}
		}
	}
	return result
}

func TestAnalyzeWorkspaceUsesStablePackageOrder(t *testing.T) {
	t.Parallel()
	packages, err := AnalyzeWorkspace("testdata/analysisworkspace")
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 2 {
		t.Fatalf("package count = %d, want 2", len(packages))
	}
	if packages[0].Path != "example.test/analysis/app" ||
		packages[1].Path != "example.test/analysis/dep" {
		t.Fatalf("package order = %q, %q", packages[0].Path, packages[1].Path)
	}
	for _, pkg := range packages {
		if pkg.Facts == nil || pkg.Files == nil || len(pkg.Sources) == 0 {
			t.Fatalf("incomplete analysis for %s", pkg.Path)
		}
	}
}

func TestSourceFactsFindShiftedDefinition(t *testing.T) {
	t.Parallel()
	files := token.NewFileSet()
	projected, err := parser.ParseFile(
		files, "sample.tgo", "package sample\ntype value struct {\n      field int\n}\n", 0,
	)
	if err != nil {
		t.Fatalf("parse projected Go: %v", err)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Instances:  make(map[*ast.Ident]types.Instance),
		Implicits:  make(map[ast.Node]types.Object),
	}
	if _, err := new(types.Config).Check(
		"sample", files, []*ast.File{projected}, info,
	); err != nil {
		t.Fatalf("check projected Go: %v", err)
	}
	parsed, err := syntax.ParseGoFile(
		files, "sample.tgo", []byte("package sample\ntype value struct {\n\tfield int\n}\n"),
		syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("parse source syntax: %v", err)
	}
	if parsed == nil {
		t.Fatal("parse source syntax returned nil")
		return
	}
	index := sourcefacts.New(parsed, info, files)
	declaration := syntax.GeneralDeclarationOf(parsed.Declarations[0])
	specification := syntax.TypeSpecificationOf(declaration.Specs[0])
	structure := syntax.StructTypeExpressionOf(specification.Type)
	name := structure.Fields.List[0].Names[0]
	object := index.DefinitionName(name)
	if object == nil || object.Name() != "field" {
		t.Fatalf("shifted field definition = %v, want field", object)
	}
}

func TestSourceFactsDoNotClassifySameLineUseAsDefinition(t *testing.T) {
	t.Parallel()
	files := token.NewFileSet()
	projected, err := parser.ParseFile(
		files,
		"sample.tgo",
		"package sample\nvar      value = 1; var copy = value\n",
		0,
	)
	if err != nil {
		t.Fatalf("parse projected Go: %v", err)
	}
	info := newInfo()
	if _, err := new(types.Config).Check(
		"sample", files, []*ast.File{projected}, info,
	); err != nil {
		t.Fatalf("check projected Go: %v", err)
	}
	definitionObject, definitionFact, useObject, useFact := sameLineIdentifierFacts(
		t, files, info,
	)
	if definitionObject == nil || useObject != definitionObject {
		t.Fatalf("objects = %v, %v", definitionObject, useObject)
	}
	if !definitionFact {
		t.Fatal("declaration is not classified as a definition")
	}
	if useFact {
		t.Fatal("same-line use is classified as a definition")
	}
}

func sameLineIdentifierFacts(
	t *testing.T,
	files *token.FileSet,
	info *types.Info,
) (types.Object, bool, types.Object, bool) {
	t.Helper()
	parsed, err := syntax.ParseGoFile(
		files,
		"sample.tgo",
		[]byte("package sample\nvar value = 1; var copy = value\n"),
		syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("parse source syntax: %v", err)
		return nil, false, nil, false
	}
	if parsed == nil {
		t.Fatal("parse source syntax returned nil")
		return nil, false, nil, false
	}
	index := sourcefacts.New(parsed, info, files)
	first := syntax.GeneralDeclarationOf(parsed.Declarations[0])
	if first == nil {
		t.Fatal("first declaration is not general")
		return nil, false, nil, false
	}
	firstValue := syntax.ValueSpecificationOf(first.Specs[0])
	if firstValue == nil {
		t.Fatal("first declaration is not a value")
		return nil, false, nil, false
	}
	definition := firstValue.Names[0]
	second := syntax.GeneralDeclarationOf(parsed.Declarations[1])
	if second == nil {
		t.Fatal("second declaration is not general")
		return nil, false, nil, false
	}
	secondValue := syntax.ValueSpecificationOf(second.Specs[0])
	if secondValue == nil {
		t.Fatal("second declaration is not a value")
		return nil, false, nil, false
	}
	use := syntax.IdentifierExpressionOf(
		secondValue.Values[0],
	)
	if use == nil {
		t.Fatal("second declaration value is not an identifier")
		return nil, false, nil, false
	}
	definitionNode := identifierNode(parsed, definition)
	useNode := identifierNode(parsed, use)
	if definitionNode == nil || useNode == nil {
		t.Fatal("identifier node is absent")
		return nil, false, nil, false
	}
	definitionObject, definitionFact := index.IdentifierFact(parsed, definitionNode)
	useObject, useFact := index.IdentifierFact(parsed, useNode)
	return definitionObject, definitionFact, useObject, useFact
}

func identifierNode(file *syntax.File, identifier *syntax.Identifier) *syntax.Node {
	var result *syntax.Node
	syntax.Inspect(file, func(node *syntax.Node) bool {
		value, ok := syntax.IdentifierOf(node)
		if ok && value.Start == identifier.Start && value.Stop == identifier.Stop {
			result = node
			return false
		}
		return result == nil
	})
	return result
}

func TestAnalysisTypeInfoOmitsGeneratedFunctionFacts(t *testing.T) {
	t.Parallel()
	generatedExpression := &ast.BasicLit{Kind: token.INT, Value: "1"}
	generatedFunction := &ast.FuncDecl{
		Name: ast.NewIdent("generated"),
		Type: &ast.FuncType{Params: new(ast.FieldList)},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ExprStmt{X: generatedExpression},
		}},
	}
	sourceExpression := &ast.BasicLit{Kind: token.INT, Value: "2"}
	generatedType := ast.NewIdent("int")
	generatedDeclaration := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{&ast.TypeSpec{
			Name: ast.NewIdent("Payload"), Type: generatedType,
		}},
	}
	info := newInfo()
	info.Types[generatedExpression] = types.TypeAndValue{Type: types.Typ[types.Int]}
	info.Types[sourceExpression] = types.TypeAndValue{Type: types.Typ[types.Int]}
	info.Types[generatedType] = types.TypeAndValue{Type: types.Typ[types.Int]}
	unit := &packageUnit{
		info: info,
		generated: map[ast.Decl]bool{
			generatedFunction:    true,
			generatedDeclaration: true,
		},
	}

	filtered := analysisTypeInfo(unit)

	if _, ok := filtered.Types[generatedExpression]; ok {
		t.Fatal("generated function expression remains in analysis facts")
	}
	if _, ok := filtered.Types[sourceExpression]; !ok {
		t.Fatal("source expression is missing from analysis facts")
	}
	if _, ok := filtered.Types[generatedType]; !ok {
		t.Fatal("generated payload type is missing from analysis facts")
	}
}
