package names

import "testing"

func TestFreshGeneratedNames(t *testing.T) {
	allocator := &Allocator{}
	allocator.Reserve("fmt", "fmt_tgo1", "tgo3")
	for _, test := range []struct {
		source, want string
	}{
		{"fmt", "fmt_tgo2"},
		{"", "tgo4"},
		{"fmt", "fmt_tgo5"},
		{"value", "value_tgo6"},
	} {
		if got := allocator.Fresh(test.source).Name; got != test.want {
			t.Fatalf("name for %q: got %q, want %q", test.source, got, test.want)
		}
	}
}
