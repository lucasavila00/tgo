package frontend

import (
	"strings"
	"testing"
)

func TestPostfixOperatorMeaning(t *testing.T) {
	for _, test := range []struct {
		name   string
		suffix string
		wrap   bool
	}{
		{"direct", "!", false},
		{"context", "!!", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := "package example\nfunc run() error { target()" + test.suffix + "; return nil }"
			replacements := postfixReplacements("main.tgo", []byte(input), make(map[string]bool))
			if len(replacements) != 1 {
				t.Fatalf("operations: %d, want 1", len(replacements))
			}
			operation := replacements[0].record
			if operation.Wrap != test.wrap {
				t.Fatalf("%s wraps: %t, want %t", test.suffix, operation.Wrap, test.wrap)
			}
			if operation.Position.Offset != strings.Index(input, test.suffix) {
				t.Fatalf("operator position: %d", operation.Position.Offset)
			}
			source := originalSource([]byte(input))
			source.apply(replacements[0].edit)
			if !strings.Contains(source.text, "target()/*"+operation.Name+"*/;") {
				t.Fatalf("prepared source: %s", source.text)
			}
		})
	}
}
