package tgolint

import (
	"testing"
)

func TestNilParallelAssignmentKeepsValueIdentity(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		unsafe bool
	}{
		{
			name: "move saved proof",
			body: "checked := other != nil\n" +
				"value, other = other, value\n" +
				"if checked { need(value) }",
			unsafe: false,
		},
		{
			name: "keep proof on copied value",
			body: "other = value\nalias = value\n" +
				"checked := value != nil\nvalue = nil\n" +
				"if checked { need(other); need(alias) }",
			unsafe: false,
		},
		{
			name: "swap different values",
			body: "value = &Item{}\nother = nil\n" +
				"value, other = other, value\nneed(value)",
			unsafe: true,
		},
		{
			name: "swap keeps non-nil value",
			body: "value = nil\nother = &Item{}\n" +
				"value, other = other, value\nneed(value)",
			unsafe: false,
		},
	}
	for _, test := range tests {
		diagnostics, err := runNilAnalysis(
			t, "value, other, alias *Item", test.body,
		)
		if err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if got := len(diagnostics) != 0; got != test.unsafe {
			t.Fatalf(
				"%s: got %d diagnostics; want unsafe=%t",
				test.name, len(diagnostics), test.unsafe,
			)
		}
	}
}
