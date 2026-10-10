package compiler

import (
	"go/token"
	"go/types"
	"testing"
)

func TestLoweringKeepsImportedPrivateResultInferred(t *testing.T) {
	external := types.NewPackage("example.com/external", "external")
	valueName := types.NewTypeName(token.NoPos, external, "value", nil)
	value := types.NewNamed(valueName, types.Typ[types.Int], nil)
	external.Scope().Insert(valueName)
	external.Scope().Insert(types.NewFunc(
		token.NoPos,
		external,
		"Factory",
		types.NewSignatureType(
			nil, nil, nil, types.NewTuple(),
			types.NewTuple(types.NewVar(token.NoPos, external, "", value)), false,
		),
	))
	external.Scope().Insert(types.NewFunc(
		token.NoPos,
		external,
		"Consume",
		types.NewSignatureType(
			nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, external, "", value),
				types.NewVar(token.NoPos, external, "", types.Typ[types.Int]),
			),
			types.NewTuple(types.NewVar(token.NoPos, external, "", types.Typ[types.Int])),
			false,
		),
	))
	external.MarkComplete()

	tests := []struct {
		name       string
		expression string
	}{
		{name: "call", expression: "external.Factory()"},
		{name: "binary", expression: "external.Factory() + external.Factory()"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, problems := Compile(PackageInput{
				Path: "sample",
				Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

import "example.com/external"

func load() (int, error) {
	return 7, nil
}

func use() (int, error) {
	return external.Consume(` + test.expression + `, load()!!), nil
}
`)}},
				FileSet: token.NewFileSet(),
				Importer: packageImporter{
					"example.com/external": external,
				},
			})
			if len(problems) != 0 {
				t.Fatalf("compile imported private result: %v", problems[0])
			}
		})
	}
}
