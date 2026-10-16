package frontend

import (
	"go/token"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/frontend/parser"
)

// Parsed holds source syntax and the call names used in propagated error context.
type Parsed struct {
	File      *ast.File
	CallNames map[*ast.PropagateExpr]string
}

// ParseFile reads native TGo syntax using the physical file and logical source name.
func ParseFile(fset *token.FileSet, filename, sourceName string, input []byte) (*Parsed, error) {
	file, err := parser.ParseTGoFile(fset, filename, sourceName, input,
		parser.ParseComments|parser.AllErrors|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	return &Parsed{File: file, CallNames: sourceCallNames(fset, file, input)}, nil
}
