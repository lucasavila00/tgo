package lowering

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"
)

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
			rewrite := newRewrite(source)
			evaluated := rewrite.expression(expression)
			saved := rewrite.save(evaluated.values[0])
			if len(saved.values) != len(test.types) {
				t.Fatalf("got %d values, want %d", len(saved.values), len(test.types))
			}
			for i, value := range saved.values {
				if value.typ == nil || value.typ.String() != test.types[i] {
					t.Errorf("value %d has type %v, want %s", i, value.typ, test.types[i])
				}
				if value.original != expression {
					t.Errorf("value %d lost its source expression", i)
				}
			}
		})
	}
}
