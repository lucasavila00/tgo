package sourceanalysis

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"tgo/pkg/syntax"
)

func TestAnalysisOwnersCoverGeneratedPublicSurface(t *testing.T) {
	t.Parallel()
	packages, err := AnalyzeWorkspace("testdata/analysisworkspace")
	if err != nil {
		t.Fatal(err)
	}
	var analysis *Package
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
	for _, name := range []string{"Choice", "ChoiceOne", "ChoiceTwo"} {
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
	analysis *Package,
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
