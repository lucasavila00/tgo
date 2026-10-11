package driver

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucasavila00/tgo/internal/checker"
	"github.com/lucasavila00/tgo/internal/lowering"
	"golang.org/x/tools/go/packages"
)

// selectedSources carries Go build metadata to source checking.
type selectedSources struct {
	module   *preparedModule
	packages []*packages.Package
}

// checkedModule carries source types and active file names to lowering.
type checkedModule struct {
	sources     *selectedSources
	packages    []*checker.Package
	activeFiles map[string]bool
}

func selectSources(module *preparedModule) *selectedSources {
	roots, err := packages.Load(&packages.Config{Dir: module.dir, Tests: true, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedExportFile | packages.NeedModule}, "./...")
	must(err)
	var selected []*packages.Package
	packages.Visit(roots, nil, func(pkg *packages.Package) { selected = append(selected, pkg) })
	return &selectedSources{module: module, packages: selected}
}

// checkSources checks all selected source packages before output changes any file.
func checkSources(sources *selectedSources) *checkedModule {
	module := sources.module
	loader, err := checker.NewLoader(module.dir, sources.packages, module.tgoFiles, module.used)
	must(err)
	checked := &checkedModule{sources: sources, activeFiles: make(map[string]bool)}
	for _, metadata := range sources.packages {
		if metadata.Module == nil || !metadata.Module.Main {
			continue
		}
		pkg, err := loader.Check(metadata)
		must(err)
		checked.packages = append(checked.packages, pkg)
		for _, filename := range pkg.GoFiles {
			name, err := filepath.Rel(module.dir, filename)
			must(err)
			if module.tgoFiles[name] != "" {
				checked.activeFiles[name] = true
			}
		}
	}
	return checked
}

// lowerModule writes ordinary Go from the checked source syntax and types.
func lowerModule(module *checkedModule) {
	for _, pkg := range module.packages {
		if len(pkg.Operations) == 0 {
			continue
		}
		rejectUnsafePropagation(pkg)
		operations := make(map[string]lowering.Propagation)
		for name, operation := range pkg.Operations {
			operations[name] = lowering.Propagation{Wrap: operation.Wrap, CallName: operation.CallName, Position: operation.Position}
		}
		source, err := lowering.SourceFromCheckedPackage(pkg, module.sources.module.dir)
		must(err)
		output, err := source.LowerPropagations(operations)
		must(err)
		for name, content := range output {
			must(os.WriteFile(name, content, 0600))
		}
	}
}

func checkGenerated(dir string) {
	for _, pkg := range loadProject(dir) {
		if len(pkg.Errors) != 0 {
			panic(fmt.Sprintf("TGo: generated Go: %s", pkg.Errors[0]))
		}
	}
}
