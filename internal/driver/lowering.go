package driver

import (
	"path/filepath"

	"github.com/lucasavila00/tgo/internal/lowering"
)

// generatedModule is the complete set of disk changes after source lowering.
type generatedModule struct {
	files    map[string][]byte
	excluded []string
}

// lowerModule builds all output contents before publication changes the disk.
func lowerModule(module *checkedModule) generatedModule {
	output := generatedModule{files: make(map[string][]byte)}
	for _, pkg := range module.packages {
		rejectUnsafePropagation(pkg)
		source, err := lowering.SourceFromCheckedPackage(pkg, module.root)
		must(err)
		files, err := source.LowerPropagations()
		must(err)
		for path, content := range files {
			relative, err := filepath.Rel(module.root, path)
			must(err)
			if module.tgoFiles[relative] != "" {
				output.files[path] = content
			}
		}
	}
	for name := range module.tgoFiles {
		if !module.activeFiles[name] {
			output.excluded = append(output.excluded, filepath.Join(module.root, name))
		}
	}
	return output
}
