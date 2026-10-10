package compiler

import (
	"go/token"
	"go/types"
	"testing"
)

func TestLoweringPlaceCapturesProducedContainerTypes(t *testing.T) {
	library := types.NewPackage("example.com/unused", "unused")
	library.MarkComplete()
	plan := buildForeignContractPlan(t, library, `package sample

func getSlice() ([]int, error) { return nil, nil }
func getMap() (map[string]int, error) { return nil, nil }
func getPointer() (*int, error) { return nil, nil }

func use(index int, key string, value int) error {
	getSlice()!![index] = value
	getMap()!![key] = value
	*getPointer()!! = value
	return nil
}
`)

	expected := map[plannedPlaceKind]types.Type{
		planSliceIndexPlace: types.NewSlice(types.Typ[types.Int]),
		planMapIndexPlace:   types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
		planDerefPlace:      types.NewPointer(types.Typ[types.Int]),
	}
	seen := make(map[plannedPlaceKind]bool)
	for _, operation := range plan.root.operations {
		if len(operation.places) != 1 {
			continue
		}
		place := operation.places[0]
		want := expected[place.kind]
		if want == nil {
			continue
		}
		seen[place.kind] = true
		assertCapturedPlaceType(t, plan, place, want)
	}
	for kind := range expected {
		if !seen[kind] {
			t.Fatalf("plan does not contain assignment place kind %d", kind)
		}
	}
}

func TestLoweringPlaceCapturesNamedForeignContainerTypes(t *testing.T) {
	library := types.NewPackage("example.com/hidden", "hidden")
	hiddenInt := newNamedContractType(library, "hiddenInt", types.Typ[types.Int])
	values := newNamedContractType(library, "Values", types.NewSlice(hiddenInt))
	hiddenKey := newNamedContractType(library, "hiddenKey", types.Typ[types.String])
	hiddenValue := newNamedContractType(library, "hiddenValue", types.Typ[types.Int])
	lookup := newNamedContractType(
		library,
		"Lookup",
		types.NewMap(hiddenKey, hiddenValue),
	)
	insertContractFunction(library, "SliceFactory", values, contractErrorType())
	insertContractFunction(library, "MapFactory", lookup, contractErrorType())
	insertContractFunction(library, "Key", hiddenKey)
	insertContractFunction(library, "Value", hiddenValue, contractErrorType())
	library.MarkComplete()

	plan := buildForeignContractPlan(t, library, `package sample

import "example.com/hidden"

func use() error {
	hidden.SliceFactory()!![0]++
	hidden.MapFactory()!![hidden.Key()]++
	hidden.MapFactory()!!["key"] = hidden.Value()!!
	return nil
}
`)
	expected := map[plannedPlaceKind]types.Type{
		planSliceIndexPlace: values,
		planMapIndexPlace:   lookup,
	}
	seen := make(map[plannedPlaceKind]bool)
	for _, operation := range plan.root.operations {
		if len(operation.places) != 1 {
			continue
		}
		place := operation.places[0]
		want := expected[place.kind]
		if want == nil {
			continue
		}
		seen[place.kind] = true
		assertCapturedPlaceType(t, plan, place, want)
	}
	for kind := range expected {
		if !seen[kind] {
			t.Fatalf("plan does not contain named assignment place kind %d", kind)
		}
	}
}

func TestLoweringRetainsForeignPrivateMapKeyContext(t *testing.T) {
	library := types.NewPackage("example.com/hidden", "hidden")
	hiddenKey := newNamedContractType(library, "hiddenKey", types.Typ[types.String])
	hiddenValue := newNamedContractType(library, "hiddenValue", types.Typ[types.Int])
	lookup := newNamedContractType(library, "Lookup", types.NewMap(hiddenKey, hiddenValue))
	insertContractFunction(library, "MapFactory", lookup, contractErrorType())
	insertContractFunction(library, "Value", hiddenValue, contractErrorType())
	library.MarkComplete()

	_, problems := Compile(PackageInput{
		Path: "sample", FileSet: token.NewFileSet(),
		Importer: packageImporter{"example.com/hidden": library},
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

import "example.com/hidden"

func use() error {
	hidden.MapFactory()!!["key"] = hidden.Value()!!
	return nil
}
`)}},
	})
	if len(problems) != 0 {
		t.Fatalf("compile private map key context: %v", problems[0])
	}
}

func newNamedContractType(
	pkg *types.Package,
	name string,
	underlying types.Type,
) *types.Named {
	typeName := types.NewTypeName(token.NoPos, pkg, name, nil)
	named := types.NewNamed(typeName, underlying, nil)
	pkg.Scope().Insert(typeName)
	return named
}

func insertContractFunction(
	pkg *types.Package,
	name string,
	results ...types.Type,
) {
	variables := make([]*types.Var, 0, len(results))
	for _, result := range results {
		variables = append(variables, types.NewVar(token.NoPos, pkg, "", result))
	}
	pkg.Scope().Insert(types.NewFunc(
		token.NoPos,
		pkg,
		name,
		types.NewSignatureType(
			nil, nil, nil, types.NewTuple(), types.NewTuple(variables...), false,
		),
	))
}

func contractErrorType() types.Type {
	return types.Universe.Lookup("error").Type()
}

func assertCapturedPlaceType(
	t *testing.T,
	plan *functionLoweringPlan,
	place *plannedPlace,
	want types.Type,
) {
	t.Helper()
	if len(place.values) == 0 || place.values[0].id == 0 {
		t.Fatalf("place does not capture its produced container: %#v", place)
	}
	captured := place.values[0]
	planned := plan.values[captured.id-1]
	if !types.Identical(captured.typ, want) || !types.Identical(planned.typ, want) {
		t.Fatalf("place captured wrong container type: place=%v plan=%v want=%v",
			captured.typ, planned.typ, want)
	}
}
