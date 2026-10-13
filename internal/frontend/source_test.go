package frontend

import (
	"reflect"
	"testing"
)

func TestSourceEditsKeepInputPositions(t *testing.T) {
	source := originalSource([]byte("abc def ghi"))
	source.apply(
		sourceEdit{start: 1, end: 3, text: "BC!", offsets: []int{1, 2, 2}},
		sourceEdit{start: 8, end: 11, text: "G", offsets: []int{8}},
	)
	if source.text != "aBC! def G" {
		t.Fatalf("prepared text: %q", source.text)
	}
	if want := []int{0, 1, 2, 2, 3, 4, 5, 6, 7, 8, 11}; !reflect.DeepEqual(source.offsets, want) {
		t.Fatalf("input offsets: %v; want %v", source.offsets, want)
	}
	// A later edit refers to the prepared text, but keeps input positions.
	source.apply(sourceEdit{start: 5, end: 8, text: "D", offsets: []int{4}})
	if want := []int{0, 1, 2, 2, 3, 4, 7, 8, 11}; !reflect.DeepEqual(source.offsets, want) {
		t.Fatalf("input offsets after second edit: %v; want %v", source.offsets, want)
	}
}
