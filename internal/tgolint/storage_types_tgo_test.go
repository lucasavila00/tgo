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
	if got := projectStorageEffects(effects, nil, projection, nil, true); len(got) != 0 {
		t.Fatalf("compound projected effects = %#v", got)
	}
}

func TestStorageEffectProjectionKeepsPrimitiveEffect(t *testing.T) {
	t.Parallel()
	effects := []GenericEffect{{TypeParameter: 0}}
	projection := []StorageEffectType{{Kind: storageTypeParameter, Parameter: 1}}
	got := projectStorageEffects(effects, nil, projection, nil, true)
	if len(got) != 1 || got[0].Receiver || got[0].TypeParameter != 1 {
		t.Fatalf("primitive projected effects = %#v", got)
	}
}

func TestStorageEffectProjectionKeepsPositiveArrayAndStructEffects(t *testing.T) {
	t.Parallel()
	effects := []GenericEffect{{TypeParameter: 0}}
	parameter := StorageEffectType{Kind: storageTypeParameter, Parameter: 1}
	projections := []StorageEffectType{
		{Kind: storageTypeArray, Length: 2, Element: []StorageEffectType{parameter}},
		{Kind: storageTypeStruct, Fields: []StorageEffectType{parameter}},
		{Kind: storageTypeNamed, Underlying: []StorageEffectType{{
			Kind: storageTypeStruct, Fields: []StorageEffectType{parameter},
		}}},
	}
	for _, projection := range projections {
		got := projectStorageEffects(effects, nil, []StorageEffectType{projection}, nil, true)
		if len(got) != 1 || got[0].TypeParameter != 1 {
			t.Fatalf("invalid-zero projected effects = %#v", got)
		}
	}
}

func TestStorageEffectProjectionDropsZeroArrayEffect(t *testing.T) {
	t.Parallel()
	effects := []GenericEffect{{TypeParameter: 0}}
	projection := []StorageEffectType{{
		Kind: storageTypeArray, Length: 0,
		Element: []StorageEffectType{{Kind: storageTypeParameter, Parameter: 0}},
	}}
	if got := projectStorageEffects(effects, nil, projection, nil, true); len(got) != 0 {
		t.Fatalf("zero-array projected effects = %#v", got)
	}
}

func TestStorageEffectProjectionKeepsReceiverEffect(t *testing.T) {
	t.Parallel()
	effects := []GenericEffect{{Receiver: true, TypeParameter: 0}}
	receiver := []StorageEffectType{{Kind: storageTypeReceiver, Parameter: 1}}
	got := projectStorageEffects(effects, receiver, nil, nil, true)
	if len(got) != 1 || !got[0].Receiver || got[0].TypeParameter != 1 {
		t.Fatalf("receiver projected effects = %#v", got)
	}
}

func TestStorageProjectedNamedTypesHaveStableEquality(t *testing.T) {
	t.Parallel()
	projection := StorageEffectType{
		Kind: storageTypeNamed, Package: "example.test/model", Name: "Box", PackageLevel: true,
		Arguments: []StorageEffectType{{Kind: storageTypeParameter, Parameter: 0}},
	}
	pkg := types.NewPackage("example.test/model", "model")
	named := types.NewNamed(types.NewTypeName(0, pkg, "Box", nil), types.NewStruct(nil, nil), nil)
	named.SetTypeParams(storageTestParameters(1))
	names := map[string]*types.Named{storageNamedTypeKey(pkg.Path(), "Box"): named}
	first := decodeStorageEffectTypeWithNames(projection, nil, []types.Type{types.Typ[types.Int]}, names)
	second := decodeStorageEffectTypeWithNames(projection, nil, []types.Type{types.Typ[types.Int]}, names)
	want, err := types.Instantiate(nil, named, []types.Type{types.Typ[types.Int]}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !storageTypesEqual(first, second) || !types.Identical(first, want) {
		t.Fatalf("projected named types differ: %s and %s", first, second)
	}
}

func TestStorageProjectedLocalNamedTypeDoesNotUsePackageDeclaration(t *testing.T) {
	t.Parallel()
	pkg := types.NewPackage("example.test/model", "model")
	packageNamed := types.NewNamed(
		types.NewTypeName(1, pkg, "Box", nil), types.NewStruct(nil, nil), nil,
	)
	projection := StorageEffectType{
		Kind: storageTypeNamed, Package: pkg.Path(), Name: "Box", Position: 2,
		Underlying: []StorageEffectType{{Kind: storageTypeBasic, Basic: int(types.Int)}},
	}
	names := map[string]*types.Named{storageNamedTypeKey(pkg.Path(), "Box"): packageNamed}
	got := decodeStorageEffectTypeWithNames(projection, nil, nil, names)
	if types.Identical(got, packageNamed) || got.Underlying().String() != "int" {
		t.Fatalf("local projected type = %s (%s)", got, got.Underlying())
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
