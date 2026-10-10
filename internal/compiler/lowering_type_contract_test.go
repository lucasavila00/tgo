package compiler

import (
	"go/token"
	"go/types"
	"testing"
)

func TestLoweringKeepsGenericAliasTypeBeforePropagation(t *testing.T) {
	_, problems := Compile(PackageInput{
		Path: "genericaliascontract",
		Sources: []File{{Name: "generic_alias_contract.tgo", Data: []byte(`package genericaliascontract

type Base[T any] bool
type Flag[T any] = Base[T]

func consume[V any](value Flag[V], number int) int {
	if value {
		return number
	}
	return 0
}

func load() (int, error) {
	return 7, nil
}

func use[V any](left int, right int) (int, error) {
	type Flag bool
	_ = Flag(false)
	return consume[V](left < right, load()!!), nil
}
`)}},
		FileSet: token.NewFileSet(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile generic alias contract: %v", problems[0])
	}
}

func TestLoweringKeepsImportedPrivateChannelResultInferred(t *testing.T) {
	external := types.NewPackage("example.com/external", "external")
	hidden := types.NewVar(token.NoPos, external, "hidden", types.Typ[types.Int])
	structure := types.NewStruct([]*types.Var{hidden}, []string{""})
	channel := types.NewChan(types.SendRecv, types.NewPointer(structure))
	external.Scope().Insert(types.NewFunc(
		token.NoPos,
		external,
		"Channel",
		types.NewSignatureType(
			nil, nil, nil, types.NewTuple(),
			types.NewTuple(
				types.NewVar(token.NoPos, external, "", channel),
				types.NewVar(token.NoPos, external, "", types.Universe.Lookup("error").Type()),
			),
			false,
		),
	))
	external.MarkComplete()

	_, problems := Compile(PackageInput{
		Path: "privatechannelcontract",
		Sources: []File{{
			Name: "private_channel_contract.tgo",
			Data: []byte(`package privatechannelcontract

import "example.com/external"

func use() error {
	select {
	case external.Channel()!! <- nil:
	default:
	}
	return nil
}
`),
		}},
		FileSet: token.NewFileSet(),
		Importer: packageImporter{
			"example.com/external": external,
		},
	})
	if len(problems) != 0 {
		t.Fatalf("compile imported private channel result: %v", problems[0])
	}
}
