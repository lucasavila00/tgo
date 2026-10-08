// Package compiler compiles tgo packages to Go source.
package compiler

import (
	"go/ast"
	"go/token"
)

type lexeme struct {
	kind  token.Token
	text  string
	start int
	end   int
}

type edit struct {
	start int
	end   int
	text  string
}

type field struct {
	Name    string
	Type    string
	Default string
}

type variant struct {
	Name   string
	Fields []field
}

type model struct {
	Name      string
	Base      string
	Predicate string
	Variants  []variant
	Fields    []field
}

type source struct {
	Name   string
	File   *ast.File
	Models []*model
}

func (m *model) requiresConstructor() bool {
	return len(m.Variants) > 0 || m.Predicate != ""
}
