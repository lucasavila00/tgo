package driver

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/packages"
)

// writeOutputs replaces selected output and removes excluded generated files.
func writeOutputs(output generatedModule) {
	for path, content := range output.files {
		must(os.WriteFile(path, content, 0644))
	}
	for _, path := range output.excluded {
		must(os.Remove(path))
	}
}

func checkGenerated(dir string) {
	for _, pkg := range loadProject(dir) {
		if len(pkg.Errors) != 0 {
			panic(fmt.Sprintf("TGo: generated Go: %s", pkg.Errors[0]))
		}
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
