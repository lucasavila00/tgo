// Package format tests formatter alignment planning.
package format

import (
	"reflect"
	"testing"
)

func TestAlignmentColumns(t *testing.T) {
	rows := []alignmentRow{
		{breakBefore: false, cells: []int{8, 7, 3}},
		{breakBefore: false, cells: []int{9}},
		{breakBefore: false, cells: []int{3, 0, 3}},
		{breakBefore: true, cells: []int{20, 1}},
		{breakBefore: false, cells: []int{2, 1}},
	}
	want := [][]int{
		{9, 17},
		nil,
		{4, 4},
		{21},
		{21},
	}
	if got := alignmentColumns(rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("alignment columns = %v, want %v", got, want)
	}
}

func TestAlignmentColumnsDiscardEmptyColumn(t *testing.T) {
	rows := []alignmentRow{
		{breakBefore: false, cells: []int{2, 0, 3}},
		{breakBefore: false, cells: []int{3, 0, 1}},
	}
	want := [][]int{{4, 4}, {4, 4}}
	if got := alignmentColumns(rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("alignment columns = %v, want %v", got, want)
	}
}
