package compiler

import (
	"go/token"
	"testing"
)

func TestParseSourceDerivesLoweredFromProjectionEdits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		source  string
		lowered bool
	}{
		{
			name:    "ordinary struct",
			source:  "package sample\n\ntype Item struct { Value int }\n",
			lowered: false,
		},
		{
			name: "successful return",
			source: "package sample\n\n" +
				"func value() (int, error) { return 1, }\n",
			lowered: true,
		},
		{
			name: "non-nil struct field",
			source: "package sample\n\n" +
				"type Item struct { Value %int }\n",
			lowered: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			source, err := parseSource(
				token.NewFileSet(), "sample.tgo", []byte(test.source),
			)
			if err != nil {
				t.Fatal(err)
			}
			if source.Lowered != test.lowered {
				t.Fatalf("Lowered = %t, want %t", source.Lowered, test.lowered)
			}
		})
	}
}
