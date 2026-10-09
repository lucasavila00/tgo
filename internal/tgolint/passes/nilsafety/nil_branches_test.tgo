package nilsafety

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
			unsafe: false,
		},
		{
			name: "later check removes alternative",
			body: "if (alias != nil || value != nil) && alias == nil {" +
				" need(value) }",
			unsafe: false,
		},
		{
			name: "saved alternatives",
			body: "checked := alias != nil || value != nil\n" +
				"if checked && alias == nil { need(value) }",
			unsafe: false,
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
			unsafe: false,
		},
		{
			name: "saved proof follows surviving alias",
			body: "alias = nil\nvalue = other\nalias = value\n" +
				"value = alias\nvalue = other\nvalue = other\n" +
				"other = &Item{}\nchecked := value != nil\n" +
				"value = &Item{}\nif checked { need(alias) }",
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
