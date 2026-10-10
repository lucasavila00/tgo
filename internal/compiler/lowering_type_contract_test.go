package compiler

import (
	"go/token"
	"go/types"
	"testing"
)

func TestLoweringKeepsGenericAliasTypeBeforePropagation(t *testing.T) {
	tests := []struct {
		name         string
		declarations string
		constraint   string
	}{
		{
			name: "any alias",
			declarations: `type Base[T any] bool
type Flag[T any] = Base[T]`,
			constraint: "any",
		},
		{
			name: "union alias",
			declarations: `type Base[T ~int | ~int64] bool
type Flag[T ~int | ~int64] = Base[T]`,
			constraint: "~int | ~int64",
		},
		{
			name:         "union named",
			declarations: "type Flag[T ~int | ~int64] bool",
			constraint:   "~int | ~int64",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, problems := Compile(PackageInput{
				Path: "genericaliascontract",
				Sources: []File{{
					Name: "generic_alias_contract.tgo",
					Data: []byte(`package genericaliascontract

` + test.declarations + `

func consume[V ` + test.constraint + `](value Flag[V], number int) int {
	if value {
		return number
	}
	return 0
}

func load() (int, error) {
	return 7, nil
}

func use[V ` + test.constraint + `](left int, right int) (int, error) {
	type Flag bool
	_ = Flag(false)
	return consume[V](left < right, load()!!), nil
}
	`),
				}},
				FileSet: token.NewFileSet(),
			})
			if len(problems) != 0 {
				t.Fatalf("compile generic alias contract: %v", problems[0])
			}
		})
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

func TestLoweringKeepsForeignPrivateConstraintIdentity(t *testing.T) {
	library := types.NewPackage("example.com/lib", "lib")
	concreteName := types.NewTypeName(token.NoPos, library, "Concrete", nil)
	concrete := types.NewNamed(concreteName, types.NewStruct(nil, nil), nil)
	library.Scope().Insert(concreteName)
	receiver := types.NewVar(token.NoPos, library, "", concrete)
	concrete.AddMethod(types.NewFunc(
		token.NoPos,
		library,
		"private",
		types.NewSignatureType(receiver, nil, nil, types.NewTuple(), types.NewTuple(), false),
	))

	privateMethod := types.NewFunc(
		token.NoPos,
		library,
		"private",
		types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false),
	)
	constraint := types.NewInterfaceType([]*types.Func{privateMethod}, nil)
	constraint.Complete()
	parameterName := types.NewTypeName(token.NoPos, library, "T", nil)
	parameter := types.NewTypeParam(parameterName, constraint)
	boxName := types.NewTypeName(token.NoPos, library, "Box", nil)
	box := types.NewNamed(boxName, types.Typ[types.Bool], nil)
	box.SetTypeParams([]*types.TypeParam{parameter})
	library.Scope().Insert(boxName)
	instantiatedBox, err := types.Instantiate(
		nil,
		box,
		[]types.Type{concrete},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	library.Scope().Insert(types.NewFunc(
		token.NoPos,
		library,
		"Consume",
		types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(
				types.NewVar(token.NoPos, library, "", instantiatedBox),
				types.NewVar(token.NoPos, library, "", types.Typ[types.Int]),
			),
			types.NewTuple(types.NewVar(token.NoPos, library, "", types.Typ[types.Int])),
			false,
		),
	))
	library.MarkComplete()

	_, problems := Compile(PackageInput{
		Path: "privateconstraintcontract",
		Sources: []File{{
			Name: "private_constraint_contract.tgo",
			Data: []byte(`package privateconstraintcontract

import . "example.com/lib"

func load() (int, error) {
	return 7, nil
}

func use(left int, right int) (int, error) {
	type Box bool
	_ = Box(false)
	return Consume(left < right, load()!!), nil
}

`),
		}},
		FileSet: token.NewFileSet(),
		Importer: packageImporter{
			"example.com/lib": library,
		},
	})
	if len(problems) != 0 {
		t.Fatalf("compile foreign private constraint: %v", problems[0])
	}
}

func TestLoweringKeepsForeignHiddenBooleanContext(t *testing.T) {
	library := types.NewPackage("example.com/hidden", "hidden")
	hiddenName := types.NewTypeName(token.NoPos, library, "hiddenBool", nil)
	hidden := types.NewNamed(hiddenName, types.Typ[types.Bool], nil)
	library.Scope().Insert(hiddenName)
	library.Scope().Insert(types.NewFunc(
		token.NoPos, library, "Consume",
		types.NewSignatureType(
			nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, library, "", hidden),
				types.NewVar(token.NoPos, library, "", types.Typ[types.Int]),
			),
			types.NewTuple(), false,
		),
	))
	library.Scope().Insert(types.NewFunc(
		token.NoPos, library, "Check",
		types.NewSignatureType(
			nil, nil, nil, types.NewTuple(),
			types.NewTuple(
				types.NewVar(token.NoPos, library, "", hidden),
				types.NewVar(
					token.NoPos, library, "", types.Universe.Lookup("error").Type(),
				),
			),
			false,
		),
	))
	library.MarkComplete()

	_, problems := Compile(PackageInput{
		Path: "hiddenbooleancontract",
		Sources: []File{{
			Name: "hidden_boolean_contract.tgo",
			Data: []byte(`package hiddenbooleancontract

import "example.com/hidden"

func left() int { return 1 }
func right() int { return 2 }
func load() (int, error) { return 7, nil }

func direct() error {
	hidden.Consume(left() < right(), load()!!)
	return nil
}

func parenthesized() error {
	hidden.Consume((left() < right()), load()!!)
	return nil
}

func nestedParentheses() error {
	hidden.Consume((((left() < right()))), load()!!)
	return nil
}

func negated() error {
	hidden.Consume(!(left() < right()), load()!!)
	return nil
}

func logicalAnd() error {
	hidden.Consume(left() < right() && right() > left(), load()!!)
	return nil
}

func logicalOr() error {
	hidden.Consume(left() > right() || right() > left(), load()!!)
	return nil
}

func parenthesizedLogical() error {
	hidden.Consume((left() < right() && right() > left()), load()!!)
	return nil
}

func logicalChain() error {
	hidden.Consume(left() < right() && right() > left() && left() < right(), load()!!)
	return nil
}

func propagatedLogicalRHS() error {
	hidden.Consume(left() < right() && hidden.Check()!!, load()!!)
	return nil
}
`),
		}},
		FileSet: token.NewFileSet(),
		Importer: packageImporter{
			"example.com/hidden": library,
		},
	})
	if len(problems) != 0 {
		t.Fatalf("compile foreign hidden boolean context: %v", problems[0])
	}
}
