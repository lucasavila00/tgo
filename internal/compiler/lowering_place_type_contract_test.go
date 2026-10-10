package compiler

import (
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
