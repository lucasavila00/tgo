package frontend

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/frontend/parser"
)

// Parsed contains Go syntax and records for added TGo forms.
type Parsed struct {
	File           *ast.File
	Comprehensions map[string]*Comprehension
	Conditionals   map[string]token.Position
	Propagations   map[string]Marker
	ShortReturns   map[string]ShortReturn
}

// ParseFile reads TGo grammar and builds the syntax and records used by checking.
// Source names are saved before checker calls replace postfix operands.
func ParseFile(fset *token.FileSet, filename, sourceName string, input []byte, used map[string]bool) (*Parsed, error) {
	file, forms, err := parser.ParseTGoFile(fset, filename, sourceName, input,
		parser.ParseComments|parser.AllErrors|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	reserveIdentifiers(file, used)
	result := &Parsed{
		File: file, Comprehensions: make(map[string]*Comprehension),
		Conditionals: make(map[string]token.Position), Propagations: make(map[string]Marker),
		ShortReturns: make(map[string]ShortReturn),
	}
	bindings := result.bindPostfixCalls(fset, forms, input, used)
	result.nameForms(fset, forms, used)
	normalizePostfix(file, bindings)
	return result, nil
}

// reserveIdentifiers adds source identifiers to used before helper names are assigned.
func reserveIdentifiers(file *ast.File, used map[string]bool) {
	ast.Inspect(file, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok && identifier.Name != "" {
			used[identifier.Name] = true
		}
		return true
	})
}
