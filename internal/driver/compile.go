package driver

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// Compile translates the module's TGo files and returns file contents by module-relative path.
// It keeps other module files in the result and panics on invalid source or compiler failures.
func Compile(dir string) map[string][]byte {
	root := findModule(dir)
	source := readModule(root)
	prepared := prepareModule(source)
	metadata, err := os.MkdirTemp("", "tgo-module-metadata-")
	must(err)
	defer func() { _ = os.RemoveAll(metadata) }()
	moduleFile := filepath.Join(metadata, "go.mod")
	must(os.WriteFile(moduleFile, prepared.moduleData, 0600))
	if sums, exists := source.files["go.sum"]; exists {
		must(os.WriteFile(filepath.Join(metadata, "go.sum"), sums, 0600))
	}
	prepared.buildFlags = []string{"-modfile=" + moduleFile}
	selected := selectSources(prepared)
	checked := checkSources(selected)
	lowerModule(checked)
	checkGeneratedOverlay(prepared.dir, prepared.overlay, prepared.buildFlags)
	return collectOutput(checked)
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("TGo: %v", err))
	}
}

func loadProjectOverlay(dir string, overlay map[string][]byte, buildFlags []string) []*packages.Package {
	roots, err := packages.Load(&packages.Config{Dir: dir, Env: append(os.Environ(), "GOWORK=off"), Overlay: overlay, BuildFlags: buildFlags, Tests: true, Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps}, "./...")
	must(err)
	var result []*packages.Package
	packages.Visit(roots, nil, func(pkg *packages.Package) { result = append(result, pkg) })
	return result
}
