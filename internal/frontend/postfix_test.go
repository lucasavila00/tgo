package frontend

import (
	"strings"
	"testing"
)

func TestMarkPostfixOperatorMeaning(t *testing.T) {
	for _, test := range []struct {
		name   string
		suffix string
		wrap   bool
	}{
		{"direct", "!", false},
		{"context", "!!", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			operations := make(map[string]Marker)
			input := "package example\nfunc run() error { target()" + test.suffix + "; return nil }"
			output := MarkPostfix("main.tgo", []byte(input), make(map[string]bool), operations)
			if len(operations) != 1 {
				t.Fatalf("operations: %d, want 1", len(operations))
			}
			for name, operation := range operations {
				if operation.Wrap != test.wrap {
					t.Fatalf("%s wraps: %t, want %t", test.suffix, operation.Wrap, test.wrap)
				}
				if operation.Position.Offset != strings.Index(input, test.suffix) {
					t.Fatalf("operator position: %d", operation.Position.Offset)
				}
				if !strings.Contains(string(output), "target()/*"+name+"*/;") {
					t.Fatalf("prepared source: %s", output)
				}
			}
		})
	}
}
