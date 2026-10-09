package typefacts

import (
	"go/token"
	"go/types"
	"testing"
)

var testedGoType = Of(nil)

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
		want TypeTag
	}{
		{"nil", nil, TypeTagNil},
		{"basic", types.Typ[types.Int], TypeTagBasic},
		{"array", types.NewArray(types.Typ[types.Int], 1), TypeTagArray},
		{"slice", types.NewSlice(types.Typ[types.Int]), TypeTagSlice},
		{"struct", types.NewStruct(nil, nil), TypeTagStruct},
		{"pointer", types.NewPointer(types.Typ[types.Int]), TypeTagPointer},
		{"tuple", tuple, TypeTagTuple},
		{
			"signature",
			types.NewSignatureType(nil, nil, nil, tuple, tuple, false),
			TypeTagSignature,
		},
		{"map", types.NewMap(types.Typ[types.String], types.Typ[types.Int]), TypeTagMap},
		{"channel", types.NewChan(types.SendRecv, types.Typ[types.Int]), TypeTagChannel},
		{"interface", types.NewInterfaceType(nil, nil), TypeTagInterface},
		{"named", named, TypeTagNamed},
		{"type parameter", parameter, TypeTagTypeParameter},
		{
			"union",
			types.NewUnion([]*types.Term{types.NewTerm(true, types.Typ[types.Int])}),
			TypeTagUnion,
		},
		{
			"other",
			types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), types.Typ[types.Int]),
			TypeTagOther,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := Of(test.typ).Tag(); got != test.want {
				t.Fatalf("Of().Tag() = %v, want %v", got, test.want)
			}
			allocations := testing.AllocsPerRun(100, func() {
				testedGoType = Of(test.typ)
			})
			if allocations != 0 {
				t.Fatalf("Of() allocations = %v, want 0", allocations)
			}
		})
	}
}
