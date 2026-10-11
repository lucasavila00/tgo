package lowering

import "testing"

func TestFreshGeneratedNames(t *testing.T) {
	r := &rewrite{names: map[string]bool{"fmt": true, "fmt_tgo1": true, "tgo3": true}}
	for _, test := range []struct {
		source, want string
	}{
		{"fmt", "fmt_tgo2"},
		{"", "tgo4"},
		{"fmt", "fmt_tgo5"},
		{"value", "value_tgo6"},
	} {
		if got := r.nameFor(test.source).Name; got != test.want {
			t.Fatalf("name for %q: got %q, want %q", test.source, got, test.want)
		}
	}
}
