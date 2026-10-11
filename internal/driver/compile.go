package driver

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/packages"
)

// Compile translates the module's TGo files and returns file contents by module-relative path.
// It keeps other module files in the result and panics on invalid source or compiler failures.
func Compile(dir string) map[string][]byte {
	root := findModule(dir)
	work, err := os.MkdirTemp("", "tgo-transpiler-")
	must(err)
	defer func() { _ = os.RemoveAll(work) }()

	source := readModule(root)
	prepared := prepareModule(source, work)
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
	roots, err := packages.Load(&packages.Config{Dir: dir, Tests: true, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps}, "./...")
	must(err)
	var result []*packages.Package
	packages.Visit(roots, nil, func(pkg *packages.Package) { result = append(result, pkg) })
	return result
}
