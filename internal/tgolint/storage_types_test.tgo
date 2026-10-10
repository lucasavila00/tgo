package tgolint

import (
	"go/types"
	"testing"
)

func TestStorageTypeProjectionPreservesStructure(t *testing.T) {
	t.Parallel()
	parameters := storageTestParameters(2)
	bindings := map[*types.TypeParam]zeroParameter{
		parameters[0]: {index: 0},
		parameters[1]: {index: 1},
	}
	structure := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "Items", types.NewSlice(parameters[0]), false),
		types.NewField(0, nil, "Lookup", types.NewMap(parameters[0], types.NewPointer(parameters[1])), false),
		types.NewField(0, nil, "Values", types.NewArray(parameters[1], 3), false),
		types.NewField(0, nil, "Input", types.NewChan(types.RecvOnly, parameters[0]), false),
	}, []string{"json:\"items\"", "", "", ""})
	projection := encodeStorageEffectType(structure, bindings)
	got := decodeStorageEffectType(projection, nil, []types.Type{types.Typ[types.Int], types.Typ[types.String]})
	want := "struct{Items []int \"json:\\\"items\\\"\"; Lookup map[int]*string; Values [3]string; Input <-chan int}"
	if got.String() != want {
		t.Fatalf("projected type = %s, want %s", got, want)
	}
}

func TestStorageTypeProjectionPreservesSignatureAndNamedArguments(t *testing.T) {
	t.Parallel()
	parameters := storageTestParameters(2)
	bindings := map[*types.TypeParam]zeroParameter{
		parameters[0]: {index: 0},
		parameters[1]: {receiver: true, index: 0},
	}
	pkg := types.NewPackage("example.test/model", "model")
	named := types.NewNamed(types.NewTypeName(0, pkg, "Pair", nil), types.NewStruct(nil, nil), nil)
	named.SetTypeParams(parameters)
	pair, err := types.Instantiate(nil, named, []types.Type{parameters[0], parameters[1]}, false)
	if err != nil {
		t.Fatal(err)
	}
	signature := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "", pair)),
		types.NewTuple(types.NewVar(0, nil, "", types.NewPointer(parameters[1]))), false)
	projection := encodeStorageEffectType(signature, bindings)
	got := decodeStorageEffectType(projection, []types.Type{types.Typ[types.String]}, []types.Type{types.Typ[types.Int]})
	if got.String() != "func(example.test/model.Pair[int, string]) *string" {
		t.Fatalf("projected signature = %s", got)
	}
}

func TestStorageEffectProjectionDropsCompoundEffect(t *testing.T) {
	t.Parallel()
	effects := []GenericEffect{{TypeParameter: 0}}
	projection := []StorageEffectType{{Kind: storageTypeSlice,
		Element: []StorageEffectType{{Kind: storageTypeParameter, Parameter: 0}}}}
	if got := projectStorageEffects(effects, nil, projection, nil); len(got) != 0 {
		t.Fatalf("compound projected effects = %#v", got)
	}
}

func TestStorageEffectProjectionKeepsPrimitiveEffect(t *testing.T) {
	t.Parallel()
	effects := []GenericEffect{{TypeParameter: 0}}
	projection := []StorageEffectType{{Kind: storageTypeParameter, Parameter: 1}}
	got := projectStorageEffects(effects, nil, projection, nil)
	if len(got) != 1 || got[0].Receiver || got[0].TypeParameter != 1 {
		t.Fatalf("primitive projected effects = %#v", got)
	}
}

func TestStorageProjectedNamedTypesHaveStableEquality(t *testing.T) {
	t.Parallel()
	projection := StorageEffectType{
		Kind: storageTypeNamed, Package: "example.test/model", Name: "Box",
		Arguments: []StorageEffectType{{Kind: storageTypeParameter, Parameter: 0}},
	}
	first := decodeStorageEffectType(projection, nil, []types.Type{types.Typ[types.Int]})
	second := decodeStorageEffectType(projection, nil, []types.Type{types.Typ[types.Int]})
	if !storageTypesEqual(first, second) {
		t.Fatalf("projected named types differ: %s and %s", first, second)
	}
}

func storageTestParameters(count int) []*types.TypeParam {
	result := make([]*types.TypeParam, count)
	for index := range result {
		result[index] = types.NewTypeParam(types.NewTypeName(0, nil, "T", nil),
			types.NewInterfaceType(nil, nil).Complete())
	}
	return result
}
