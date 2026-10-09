// Package compiler compiles tgo packages to Go source.
package compiler

import (
	"go/ast"
	"go/token"

	"tgo/pkg/syntax"
)

type edit struct {
	start int
	end   int
	text  string
}

type field struct {
	Name          string
	Type          string
	Tag           string
	Default       string
	TypeLine      int
	TypeColumn    int
	DefaultLine   int
	DefaultColumn int
}

type variant struct {
	JSONName string
	Boxed    bool
	Name     string
	Fields   []field
}

type model struct {
	JSON            enumJSON
	Name            string
	Enum            bool
	Line            int
	Column          int
	Base            string
	BaseLine        int
	BaseColumn      int
	Predicate       string
	PredicateLine   int
	PredicateColumn int
	Variants        []variant
	Fields          []field
}

type source struct {
	JSONPackage    string
	FmtPackage     string
	Name           string
	Data           []byte
	Tree           *syntax.File
	File           *ast.File
	Models         []*model
	MatchMarker    string
	DefaultMarker  string
	Propagations   map[string]propagationSource
	Comprehensions map[string]comprehensionSource
	NonNil         map[token.Pos]bool
}

type propagationSource struct {
	Bang token.Pos
	Name string
}

type comprehensionSource struct {
	Position token.Pos
	Map      bool
}

// requiresConstructor reports whether a model type has an invalid zero value.
func (m *model) requiresConstructor() bool {
	return len(m.Variants) > 0 || m.Predicate != ""
}
