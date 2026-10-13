package driver

import (
	"path/filepath"

	"github.com/lucasavila00/tgo/internal/checker"
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
		if len(pkg.Operations) == 0 {
			continue
		}
		rejectUnsafePropagation(pkg)
		source, err := lowering.SourceFromCheckedPackage(pkg, module.root)
		must(err)
		files, err := source.LowerPropagations(propagationRequests(pkg))
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

// propagationRequests leaves other language forms to their lowering rules.
func propagationRequests(pkg *checker.Package) map[string]lowering.Propagation {
	operations := make(map[string]lowering.Propagation)
	for name, operation := range pkg.Operations {
		if operation.Conditional || operation.Comprehension || operation.Element || operation.ShortReturn {
			continue
		}
		operations[name] = lowering.Propagation{Wrap: operation.Wrap, CallName: operation.CallName,
			Position: operation.Position}
	}
	return operations
}
