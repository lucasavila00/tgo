package lowering

import (
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"go/ast"
	"os"
	"path/filepath"
	"testing"
)

// TestExpressionResultsKeepTypes checks saved values retain contextual, tuple, comma-ok, and void result types.
func TestExpressionResultsKeepTypes(t *testing.T) {
	dir := t.TempDir()
	input := `package example
func consume(value uint64) {}
func shift(n uint) { consume(1 << n) }
func pair() (int, string) { return 1, "x" }
func ping() {}
func run() {
	a, b := pair()
	table := map[string]int{}
	value, ok := table["x"]
	_, _, _, _ = a, b, value, ok
	ping()
}
`
	for name, content := range map[string]string{
		"go.mod":  "module example.com/results\n\ngo 1.27.1\n",
		"main.go": input,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, err := Load(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		expression string
		types      []string
	}{
		{"1 << n", []string{"uint64"}},
		{"pair()", []string{"int", "string"}},
		{`table["x"]`, []string{"int", "bool"}},
		{"ping()", nil},
	} {
		t.Run(test.expression, func(t *testing.T) {
			var expression ast.Expr
			for site, candidate := range source.inventory.Expressions {
				if input[site.Start:site.End] == test.expression {
					expression = candidate
				}
			}
			if expression == nil {
				t.Fatal("expression not found")
			}
			rewrite := newRewrite(source, shared.Input{})
			evaluated := rewrite.Expression(expression)
			saved := rewrite.Save(evaluated.Values[0])
			checkExpressionValues(t, saved.Values, expression, test.types)
		})
	}
}

// checkExpressionValues compares saved result types and verifies each result retains its source expression.
func checkExpressionValues(t *testing.T, values []shared.Value, original ast.Expr, expectedTypes []string) {
	t.Helper()
	if len(values) != len(expectedTypes) {
		t.Fatalf("got %d values, want %d", len(values), len(expectedTypes))
	}
	for i, value := range values {
		if value.Type == nil || value.Type.String() != expectedTypes[i] {
			t.Errorf("value %d has type %v, want %s", i, value.Type, expectedTypes[i])
		}
		if value.Original != original {
			t.Errorf("value %d lost its source expression", i)
		}
	}
}
