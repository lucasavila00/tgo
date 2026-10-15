package checker

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/frontend/parser"
	"os"
	"path/filepath"

	"github.com/lucasavila00/tgo/internal/frontend"
)

// packageSource keeps syntax and its added TGo forms together until registration.
type packageSource struct {
	file  *ast.File
	added *frontend.Parsed
}

// parseFile reads one source file and selects TGo or Go parsing from the loader file map.
// TGo results include the feature records needed for checking.
func (loader *Loader) parseFile(filename string) (packageSource, error) {
	input, err := os.ReadFile(filename)
	if err != nil {
		return packageSource{}, err
	}
	name, err := filepath.Rel(loader.root, filename)
	if err != nil {
		return packageSource{}, err
	}
	if sourceName := loader.tgoFiles[name]; sourceName != "" {
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
