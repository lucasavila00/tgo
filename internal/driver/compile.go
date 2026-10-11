package driver

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// Compile translates the module's TGo files into Go files in the same module.
// Invalid source and compiler failures panic.
func Compile(dir string) {
	root := findModule(dir)
	source := readModule(root)
	prepared := prepareModule(source)
	selected := selectSources(prepared)
	checked := checkSources(selected)
	lowerModule(checked)
	checkGenerated(prepared.dir)
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
