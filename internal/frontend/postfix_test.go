package frontend

import (
	"strings"
	"testing"
)

// TestPostfixOperatorMeaning checks that ! preserves the error and !! requests added context.
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
			if operation.EndOffset != operation.Position.Offset+len(test.suffix) {
				t.Fatalf("operator end: %d", operation.EndOffset)
			}
			source := originalSource([]byte(input))
			source.apply(replacements[0].edit)
			if !strings.Contains(source.text, "target()/*"+operation.Name+"*/;") {
				t.Fatalf("prepared source: %s", source.text)
			}
		})
	}
}

// TestPostfixOperatorSpans checks that adjacent bytes define !! and each edit consumes only its operator.
func TestPostfixOperatorSpans(t *testing.T) {
	for _, test := range []struct {
		name, input string
		operators   []string
	}{
		{"mixed", "outer(inner()!)!!", []string{"!", "!!"}},
		{"comment before", "call() /* operand */ !!", []string{"!!"}},
		{"space between", "call()! !", []string{"!"}},
		{"comment between", "call()!/* split */!", []string{"!"}},
		{"extra operator", "call()!!!", []string{"!!"}},
		{"not postfix", "!call()", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			replacements := postfixReplacements("main.tgo", []byte(test.input), make(map[string]bool))
			if len(replacements) != len(test.operators) {
				t.Fatalf("operators: %d, want %d", len(replacements), len(test.operators))
			}
			for index, replacement := range replacements {
				edit, record := replacement.edit, replacement.record
				if got := test.input[edit.start:edit.end]; got != test.operators[index] {
					t.Fatalf("operator %d: %q, want %q", index, got, test.operators[index])
				}
				if record.Position.Offset != edit.start || record.EndOffset != edit.end {
					t.Fatalf("record span: %d:%d; edit: %d:%d", record.Position.Offset, record.EndOffset,
						edit.start, edit.end)
				}
				if record.Wrap != (test.operators[index] == "!!") {
					t.Fatalf("operator %d wraps: %t", index, record.Wrap)
				}
			}
		})
	}
}
