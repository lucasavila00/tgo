package passes

import (
	"slices"
	"testing"
)

func TestPassOrder(t *testing.T) {
	want := []string{
		"TGo source policy",
		"validation model discovery",
		"nil safety",
		"iota modernization",
		"error return modernization",
		"success return modernization",
		"Go model use",
		"generic zero safety",
	}
	if got := PassNames(); !slices.Equal(got, want) {
		t.Fatalf("PassNames() = %v, want %v", got, want)
	}
}
