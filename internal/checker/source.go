package checker

import (
	"os"
	"path/filepath"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
	"github.com/lucasavila00/tgo/internal/frontend/parser"
	"github.com/lucasavila00/tgo/internal/sourcefile"
	"golang.org/x/tools/go/packages"
)

// packageSource keeps syntax and its added TGo forms together until registration.
type packageSource struct {
	file  *ast.File
	added *frontend.Parsed
}

// parseFile reads one source file and selects TGo or Go parsing from the loader file map.
// TGo results include the feature records needed for checking.
func (loader *Loader) parseFile(filename string) (packageSource, error) {
	sourceName := ""
	if original := loader.originalSources[filename]; original != "" {
		filename, sourceName = original, original
	}
	input, err := os.ReadFile(filename)
	if err != nil {
		return packageSource{}, err
	}
	name, err := filepath.Rel(loader.root, filename)
	if err != nil {
		return packageSource{}, err
	}
	if staged := loader.tgoFiles[name]; staged != "" {
		sourceName = staged
	}
	if sourceName != "" {
		parsed, err := frontend.ParseFile(loader.fset, filename, sourceName, input)
		if err != nil {
			return packageSource{}, err
		}
		return packageSource{file: parsed.File, added: parsed}, nil
	}
	file, err := parser.ParseFile(loader.fset, filename, input, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return packageSource{}, err
	}
	return packageSource{file: file}, nil
}

// recordParsedFile retains display text on the source nodes already present in checked syntax.
func (pkg *Package) recordParsedFile(parsed *frontend.Parsed) {
	for expression, name := range parsed.CallNames {
		pkg.CallNames[expression] = name
	}
}

// prepareSourceImports checks enums and their Go dependents from source to keep type identity.
func (loader *Loader) prepareSourceImports(metadata *packages.Package) error {
	var failure error
	packages.Visit([]*packages.Package{metadata}, func(dependency *packages.Package) bool {
		return failure == nil && !loader.preparedImports[dependency.ID]
	}, func(dependency *packages.Package) {
		if failure != nil || loader.preparedImports[dependency.ID] {
			return
		}
		var hasSource bool
		hasSource, failure = loader.findTGoSources(dependency)
		if failure == nil && (hasSource || loader.importsEnum(dependency)) {
			_, failure = loader.Check(dependency)
		}
		if failure == nil {
			loader.preparedImports[dependency.ID] = true
		}
	})
	return failure
}

// findTGoSources pairs selected dependency outputs with original inputs without changing dependency files.
func (loader *Loader) findTGoSources(metadata *packages.Package) (bool, error) {
	found := false
	filenames := metadata.CompiledGoFiles
	if len(filenames) == 0 {
		filenames = metadata.GoFiles
	}
	for _, output := range filenames {
		input := sourcefile.InputName(output)
		if input == "" {
			continue
		}
		if _, err := os.Stat(input); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return false, err
		}
		loader.originalSources[output] = input
		found = true
	}
	return found, nil
}

// declaresEnum reports whether source checking found a native enum declaration.
func (pkg *Package) declaresEnum() bool {
	for _, name := range pkg.Types.Scope().Names() {
		object, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
		if ok && types.EnumOf(object.Type()) != nil {
			return true
		}
	}
	return false
}

// importsEnum reports whether a direct dependency carries source enum type identities.
func (loader *Loader) importsEnum(metadata *packages.Package) bool {
	for _, dependency := range metadata.Imports {
		if loader.enumDependencies[dependency.ID] {
			return true
		}
	}
	return false
}
