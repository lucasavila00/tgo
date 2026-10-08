// Package compiler compiles tgo packages to Go source.
package compiler

import (
	"go/ast"
	"go/token"
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
	Name   string
	Fields []field
}

type model struct {
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
	Name          string
	Data          []byte
	File          *ast.File
	Models        []*model
	MatchMarker   string
	DefaultMarker string
	Propagations  map[string]propagationSource
}

type propagationSource struct {
	Bang token.Pos
	Name string
}

// requiresConstructor reports whether a model type has an invalid zero value.
func (m *model) requiresConstructor() bool {
	return len(m.Variants) > 0 || m.Predicate != ""
}
