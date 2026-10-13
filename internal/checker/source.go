package checker

import (
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"

	"github.com/lucasavila00/tgo/internal/checker/types"
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
		parsed, err := frontend.ParseFile(loader.fset, filename, sourceName, input, loader.used)
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

// recordParsedFile registers checker operations once, then indexes source nodes.
func (pkg *Package) recordParsedFile(parsed *frontend.Parsed) {
	for name, record := range parsed.Comprehensions {
		pkg.Operations[name] = types.TGoOperation{
			Comprehension: true, Ranges: record.Ranges, Output: record.Output, Position: record.Position,
		}
		pkg.Operations[record.Output] = types.TGoOperation{Element: true, Position: record.Position}
	}
	for name, record := range parsed.ShortReturns {
		pkg.Operations[name] = types.TGoOperation{
			ShortReturn: true, ZeroSuccess: record.ZeroSuccess, Position: record.Position,
		}
	}
	for name, position := range parsed.Conditionals {
		pkg.Operations[name] = types.TGoOperation{Conditional: true, Position: position}
	}
	for name, operation := range parsed.Propagations {
		pkg.Operations[name] = types.TGoOperation{
			Wrap: operation.Wrap, Position: operation.Position, CallName: operation.CallName,
		}
	}
	ast.Inspect(parsed.File, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.CallExpr:
			if identifier, isIdentifier := node.Fun.(*ast.Ident); isIdentifier {
				if record := parsed.Comprehensions[identifier.Name]; record != nil {
					pkg.Comprehensions[node] = record
				}
			}
		case *ast.ReturnStmt:
			if record, found := parsedShortReturn(node, parsed.ShortReturns); found {
				pkg.ShortReturns[node] = record
			}
		}
		return true
	})
}

// parsedShortReturn finds the record for a return containing one generated marker call.
func parsedShortReturn(statement *ast.ReturnStmt,
	records map[string]frontend.ShortReturn) (frontend.ShortReturn, bool) {
	if len(statement.Results) != 1 {
		return frontend.ShortReturn{}, false
	}
	call, isCall := statement.Results[0].(*ast.CallExpr)
	if !isCall {
		return frontend.ShortReturn{}, false
	}
	identifier, isIdentifier := call.Fun.(*ast.Ident)
	if !isIdentifier {
		return frontend.ShortReturn{}, false
	}
	record, found := records[identifier.Name]
	return record, found
}
