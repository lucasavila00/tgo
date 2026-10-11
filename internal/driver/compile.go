package driver

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// Compile translates the module's TGo files and returns file contents by module-relative path.
// It keeps other module files in the result and panics on invalid source or compiler failures.
func Compile(dir string) map[string][]byte {
	root := findModule(dir)
	source := readModule(root)
	prepared := prepareModule(source)
	selected := selectSources(prepared)
	checked := checkSources(selected)
	lowerModule(checked)
	checkGeneratedOverlay(prepared.dir, prepared.overlay)
	return collectOutput(checked)
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("TGo: %v", err))
	}
}

func loadProject(dir string) []*packages.Package {
	return loadProjectOverlay(dir, nil)
}

func loadProjectOverlay(dir string, overlay map[string][]byte) []*packages.Package {
	roots, err := packages.Load(&packages.Config{Dir: dir, Overlay: overlay, Tests: true, Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps}, "./...")
	must(err)
	var result []*packages.Package
	packages.Visit(roots, nil, func(pkg *packages.Package) { result = append(result, pkg) })
	return result
}
