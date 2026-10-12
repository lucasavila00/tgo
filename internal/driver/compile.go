package driver

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// Compile translates the module's TGo files into Go files in the same module.
// On failure, it restores previous generated files and panics.
func Compile(dir string) {
	root := findModule(dir)
	source := readModule(root)
	outputs := saveOutputs(source)
	complete := false
	defer func() {
		if !complete {
			if err := restoreOutputs(outputs); err != nil {
				failure := recover()
				panic(fmt.Sprintf("%v; TGo: output rollback failed: %v", failure, err))
			}
		}
	}()
	prepared := prepareModule(source)
	selected := selectSources(prepared)
	checked := checkSources(selected)
	lowerModule(checked)
	checkGenerated(prepared.dir)
	complete = true
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("TGo: %v", err))
	}
}

func loadProject(dir string) []*packages.Package {
	roots, err := packages.Load(&packages.Config{Dir: dir, Tests: true, Mode: packages.NeedName |
		packages.NeedTypes | packages.NeedImports | packages.NeedDeps}, "./...")
	must(err)
	var result []*packages.Package
	packages.Visit(roots, nil, func(pkg *packages.Package) { result = append(result, pkg) })
	return result
}
