package driver

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// Compile translates the module's TGo files and returns file contents by module-relative path.
// It writes generated Go into the module, keeps other files in the result, and panics on failure.
func Compile(dir string) map[string][]byte {
	root := findModule(dir)
	source := readModule(root)
	prepared := prepareModule(source)
	selected := selectSources(prepared)
	checked := checkSources(selected)
	lowerModule(checked)
	checkGenerated(prepared.dir)
	return collectOutput(checked)
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("TGo: %v", err))
	}
}

func loadProject(dir string) []*packages.Package {
	roots, err := packages.Load(&packages.Config{Dir: dir, Tests: true, Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps}, "./...")
	must(err)
	var result []*packages.Package
	packages.Visit(roots, nil, func(pkg *packages.Package) { result = append(result, pkg) })
	return result
}
