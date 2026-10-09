package tgolint

import (
	"testing"
)

func TestNilBranchAlternativesKeepRelations(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		unsafe bool
	}{
		{
			name: "alias alternatives",
			body: "value = other\n" +
				"if value != nil || other != nil { need(value) }",
		},
		{
			name: "later check removes alternative",
			body: "if (alias != nil || value != nil) && alias == nil {" +
				" need(value) }",
		},
		{
			name: "saved alternatives",
			body: "checked := alias != nil || value != nil\n" +
				"if checked && alias == nil { need(value) }",
		},
		{
			name:   "unresolved alternative",
			body:   "if alias != nil || value != nil { need(value) }",
			unsafe: true,
		},
		{
			name: "saved false compound guard",
			body: "value = &Item{}\nother = nil\n" +
				"checked := (other != nil || value == nil) || value == nil\n" +
				"value = nil\nif checked { need(value) }",
		},
	}
	for _, test := range tests {
		diagnostics, err := runNilAnalysis(
			"value, other, alias *Item", test.body,
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
