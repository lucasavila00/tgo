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
	analysis := (*Package)(nil)
	for _, pkg := range packages {
		if pkg.Path == "example.test/analysis/dep" && !pkg.Test {
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
	if len(packages) != 6 {
		t.Fatalf("package count = %d, want 6", len(packages))
	}
	if packages[0].Path != "example.test/analysis/app" ||
		packages[1].Path != "example.test/analysis/dep" ||
		packages[2].Path != "example.test/analysis/dep" ||
		packages[3].Path != "example.test/analysis/dep_test" ||
		packages[4].Path != "example.test/analysis/fresh" ||
		packages[5].Path != "example.test/analysis/fresh_test" {
		t.Fatalf(
			"package order = %q, %q, %q, %q, %q, %q",
			packages[0].Path, packages[1].Path,
			packages[2].Path, packages[3].Path,
			packages[4].Path, packages[5].Path,
		)
	}
	for _, pkg := range packages {
		if pkg.Facts == nil || pkg.Files == nil || len(pkg.Sources) == 0 {
			t.Fatalf("incomplete analysis for %s", pkg.Path)
		}
	}
}

func TestAnalyzeWorkspaceUsesInMemoryProductionForExternalTest(t *testing.T) {
	t.Parallel()
	packages, err := AnalyzeWorkspace("testdata/analysisworkspace")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range packages {
		if pkg.Path != "example.test/analysis/fresh_test" {
			continue
		}
		if !pkg.Test || !pkg.External || len(pkg.Sources) != 1 ||
			pkg.Sources[0].Name != "external_test.tgo" {
			t.Fatalf(
				"external test view is incomplete: test=%t external=%t sources=%d",
				pkg.Test, pkg.External, len(pkg.Sources),
			)
		}
		return
	}
	t.Fatal("source-only external test view is absent")
}

func TestAnalyzeWorkspaceKeepsPackageViewsSeparate(t *testing.T) {
	t.Parallel()
	packages, err := AnalyzeWorkspace("testdata/analysisworkspace")
	if err != nil {
		t.Fatal(err)
	}
	production := packageView(t, packages, false, false)
	internal := packageView(t, packages, true, false)
	external := packageView(t, packages, true, true)
	if production.Package.Scope().Lookup("internalTestValue") != nil {
		t.Fatal("production scope contains an internal test declaration")
	}
	if production.Package.Scope().Lookup("externalTestValue") != nil {
		t.Fatal("production scope contains an external test declaration")
	}
	if len(production.Sources) != 2 || len(internal.Sources) != 1 ||
		len(external.Sources) != 1 {
		t.Fatalf(
			"source counts = %d, %d, %d, want 2, 1, 1",
			len(production.Sources), len(internal.Sources), len(external.Sources),
		)
	}
	if internal.Sources[0].Name != "internal_test.tgo" ||
		external.Sources[0].Name != "external_test.tgo" {
		t.Fatalf(
			"test sources = %s, %s",
			internal.Sources[0].Name, external.Sources[0].Name,
		)
	}
	if internal.Package.Scope().Lookup("Value") == nil ||
		internal.Package.Scope().Lookup("internalTestValue") == nil {
		t.Fatal("internal test scope is incomplete")
	}
	if external.Package.Scope().Lookup("externalTestValue") == nil {
		t.Fatal("external test scope is incomplete")
	}
}

func packageView(
	t *testing.T,
	packages []*Package,
	test bool,
	external bool,
) *Package {
	t.Helper()
	path := "example.test/analysis/dep"
	if external {
		path += "_test"
	}
	for _, pkg := range packages {
		if pkg.Path == path && pkg.Test == test && pkg.External == external {
			return pkg
		}
	}
	t.Fatalf("package view test=%t external=%t is absent", test, external)
	return nil
}

func TestAnalyzeTestPackageLoadsEachTestView(t *testing.T) {
	tests := []struct {
		name       string
		external   bool
		wantPath   string
		wantSource string
		wantCount  int
	}{
		{
			name: "internal", external: false,
			wantPath:   "example.test/analysis/dep",
			wantSource: "internal_test.tgo", wantCount: 3,
		},
		{
			name: "external", external: true,
			wantPath:   "example.test/analysis/dep_test",
			wantSource: "external_test.tgo", wantCount: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis, err := AnalyzeTestPackage(
				"testdata/analysisworkspace",
				"example.test/analysis/dep",
				test.external,
				token.NewFileSet(),
			)
			if err != nil {
				t.Fatal(err)
			}
			if analysis == nil || analysis.Path != test.wantPath ||
				len(analysis.Sources) != test.wantCount {
				t.Fatalf("test analysis: %#v", analysis)
			}
			found := false
			for _, source := range analysis.Sources {
				if source.Name == test.wantSource {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing source %s", test.wantSource)
			}
		})
	}
}
