package driver

import (
	"path/filepath"

	"github.com/lucasavila00/tgo/internal/checker"
	"golang.org/x/tools/go/packages"
)

// selectedSources carries Go build metadata to source checking.
type selectedSources struct {
	module   *preparedModule
	packages []*packages.Package
}

// checkedModule carries source types and active file names to lowering.
type checkedModule struct {
	root        string
	tgoFiles    map[string]string
	packages    []*checker.Package
	activeFiles map[string]bool
}

func selectSources(module *preparedModule) *selectedSources {
	roots, err := packages.Load(&packages.Config{Dir: module.dir, Tests: true, Mode: packages.NeedName |
		packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps |
		packages.NeedExportFile | packages.NeedModule}, "./...")
	must(err)
	var selected []*packages.Package
	packages.Visit(roots, nil, func(pkg *packages.Package) { selected = append(selected, pkg) })
	return &selectedSources{module: module, packages: selected}
}

// checkSources checks selected source packages before lowering.
func checkSources(sources *selectedSources) *checkedModule {
	module := sources.module
	sizes, err := checker.TargetSizes(module.dir)
	must(err)
	loader := checker.NewLoader(module.dir, sources.packages, module.tgoFiles, module.used, sizes)
	checked := &checkedModule{root: module.dir, tgoFiles: module.tgoFiles, activeFiles: make(map[string]bool)}
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
