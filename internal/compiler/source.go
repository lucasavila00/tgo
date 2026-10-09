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
	JSONPackage      string
	JSONV2Package    string
	JSONTextPackage  string
	StringsPackage   string
	FmtPackage       string
	ExternalJSONTo   string
	AdjacentJSONTo   string
	Name             string
	Data             []byte
	Tree             *syntax.File
	File             *ast.File
	Models           []*model
	DefaultMarker    string
	Propagations     map[string]propagationSource
	Comprehensions   map[string]comprehensionSource
	Exhaustive       map[token.Pos]bool
	NonNil           map[token.Pos]bool
	GeneratedHelpers map[string]bool
	Lowered          bool
}

type propagationSource struct {
	Bang        token.Pos
	Name        string
	Transparent bool
}

type comprehensionSource struct {
	Position token.Pos
	Map      bool
}

// initiallyNeedsLowering reports whether parsed TGo syntax changes Go output.
func (s *source) initiallyNeedsLowering() bool {
	if len(s.Propagations) > 0 || len(s.Comprehensions) > 0 ||
		len(s.Exhaustive) > 0 || len(s.NonNil) > 0 {
		return true
	}
	for _, declaration := range s.Models {
		if declaration.Enum || declaration.Predicate != "" {
			return true
		}
		for _, field := range declaration.Fields {
			if field.Default != "" {
				return true
			}
		}
	}
	return false
}

// requiresConstructor reports whether a model type has an invalid zero value.
func (m *model) requiresConstructor() bool {
	return len(m.Variants) > 0 || m.Predicate != ""
}
