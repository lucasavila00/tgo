package tgolint

import (
	"go/token"
	"go/types"
	"testing"
)

func TestGoTypeOf(t *testing.T) {
	tuple := types.NewTuple()
	named := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "Named", nil),
		types.Typ[types.Int],
		nil,
	)
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		types.Universe.Lookup("any").Type(),
	)
	cases := []struct {
		name string
		typ  types.Type
		want goTypeTag
	}{
		{"nil", nil, goTypeTagNil},
		{"basic", types.Typ[types.Int], goTypeTagBasic},
		{"array", types.NewArray(types.Typ[types.Int], 1), goTypeTagArray},
		{"slice", types.NewSlice(types.Typ[types.Int]), goTypeTagSlice},
		{"struct", types.NewStruct(nil, nil), goTypeTagStruct},
		{"pointer", types.NewPointer(types.Typ[types.Int]), goTypeTagPointer},
		{"tuple", tuple, goTypeTagTuple},
		{
			"signature",
			types.NewSignatureType(nil, nil, nil, tuple, tuple, false),
			goTypeTagSignature,
		},
		{"map", types.NewMap(types.Typ[types.String], types.Typ[types.Int]), goTypeTagMap},
		{"channel", types.NewChan(types.SendRecv, types.Typ[types.Int]), goTypeTagChannel},
		{"interface", types.NewInterfaceType(nil, nil), goTypeTagInterface},
		{"named", named, goTypeTagNamed},
		{"type parameter", parameter, goTypeTagTypeParameter},
		{
			"union",
			types.NewUnion([]*types.Term{types.NewTerm(true, types.Typ[types.Int])}),
			goTypeTagUnion,
		},
		{
			"other",
			types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), types.Typ[types.Int]),
			goTypeTagOther,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := goTypeOf(test.typ).Tag(); got != test.want {
				t.Fatalf("goTypeOf().Tag() = %v, want %v", got, test.want)
			}
		})
	}
}
