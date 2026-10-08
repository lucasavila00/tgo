package app_test

import (
	"reflect"
	"testing"
	"unsafe"

	"example.com/business/model"
)

func TestBoxedEnumLayout(t *testing.T) {
	if size := unsafe.Sizeof(model.Large{}); size != 24 {
		t.Fatalf("boxed enum size: %d", size)
	}
	if size := unsafe.Sizeof(model.Equal{}); size != 64 {
		t.Fatalf("mixed enum size: %d", size)
	}
	layout := reflect.TypeOf(model.Equal{})
	if _, ok := layout.FieldByName("tgoFirst"); ok {
		t.Fatal("first equal-size payload must be boxed")
	}
	if _, ok := layout.FieldByName("tgoSecond"); !ok {
		t.Fatal("second equal-size payload must stay inline")
	}
	first := model.LargeFirst{Data: [64]byte{1, 2}}
	second := model.LargeSecond{Data: [64]byte{3, 4}}
	a := model.NewLargeFirst(first)
	b := model.NewLargeSecond(second)
	empty := model.NewLargeEmpty(model.LargeEmpty{})
	if a.TgoTag() != 1 || a.TgoFirst() != first {
		t.Fatal("first boxed payload")
	}
	if b.TgoTag() != 2 || b.TgoSecond() != second {
		t.Fatal("second boxed payload")
	}
	if empty.TgoTag() != 3 || empty.TgoEmpty() != (model.LargeEmpty{}) {
		t.Fatal("empty variant")
	}
	mixed := model.EqualSecond{Data: [40]byte{5}}
	if model.NewEqualSecond(mixed).TgoSecond() != mixed {
		t.Fatal("inline payload")
	}
}
