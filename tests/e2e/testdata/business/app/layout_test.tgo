package app_test

import (
	"reflect"
	"testing"
	"unsafe"

	"example.com/business/model"
)

func TestBoxedEnumLayout(t *testing.T) {
	if size := unsafe.Sizeof(*new(model.Large)); size != 24 {
		t.Fatalf("boxed enum size: %d", size)
	}
	if size := unsafe.Sizeof(*new(model.Equal)); size != 64 {
		t.Fatalf("mixed enum size: %d", size)
	}
	layout := reflect.TypeOf(*new(model.Equal))
	if _, ok := layout.FieldByName("tgoFirst"); ok {
		t.Fatal("first equal-size payload must be boxed")
	}
	if _, ok := layout.FieldByName("tgoSecond"); !ok {
		t.Fatal("second equal-size payload must stay inline")
	}
	first := model.LargeFirst{Data: [64]byte{1, 2}}
	second := model.LargeSecond{Data: [64]byte{3, 4}}
	a := first.Large()
	b := second.Large()
	empty := model.LargeEmpty{}.Large()
	if a.Tag() != model.LargeTagFirst || a.FirstPayload() != first {
		t.Fatal("first boxed payload")
	}
	if b.Tag() != model.LargeTagSecond || b.SecondPayload() != second {
		t.Fatal("second boxed payload")
	}
	if empty.Tag() != model.LargeTagEmpty || empty.EmptyPayload() != (model.LargeEmpty{}) {
		t.Fatal("empty variant")
	}
	mixed := model.EqualSecond{Data: [40]byte{5}}
	if mixed.Equal().SecondPayload() != mixed {
		t.Fatal("inline payload")
	}
	var read model.LargeFirst
	allocations := testing.AllocsPerRun(1000, func() {
		read = a.FirstPayload()
	})
	if allocations != 0 || read != first {
		t.Fatalf("boxed accessor cost: %f allocations", allocations)
	}
}

func TestEnumPublicAPI(t *testing.T) {
	namedZero := model.NamedZeroZero{}.NamedZero()
	if namedZero.Tag() != model.NamedZeroTagZero || namedZero.Tag() == 0 {
		t.Fatal("declared Zero variant tag")
	}
	inline := model.EqualFirst{}.Equal().SecondPayload()
	if inline != (model.EqualSecond{}) {
		t.Fatal("wrong inline accessor did not return its inactive slot")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("wrong boxed accessor did not panic")
		}
	}()
	_ = model.EqualSecond{}.Equal().FirstPayload()
}
