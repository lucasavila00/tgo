package compiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestFunctionNamesIncludeClosureUses(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "names.go", `package sample
func use() {
	_ = func() any { return captured }
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	function := file.Decls[0].(*ast.FuncDecl)
	signature := types.NewSignatureType(
		nil, nil, nil, types.NewTuple(), types.NewTuple(), false,
	)
	if !functionNames(function.Body, signature)["captured"] {
		t.Fatal("closure use was not reserved")
	}
}

func TestPropagationFormatImportNames(t *testing.T) {
	t.Parallel()
	body := `
var failure error
func load() (int, error) { return 1, failure }
func use() (int, error) {
	value := load()!
	return value, nil
}
`
	tests := []struct {
		name      string
		prefix    string
		want      string
		forbidden string
	}{
		{name: "new", want: `import "fmt"`, forbidden: "fmt_1.Errorf"},
		{
			name: "normal", prefix: "import \"fmt\"\n",
			want: `fmt.Errorf`,
		},
		{
			name: "custom", prefix: "import formatting \"fmt\"\n",
			want: `formatting.Errorf`,
		},
		{
			name: "blank", prefix: "import _ \"fmt\"\n",
			want: `import "fmt"`, forbidden: `import _ "fmt"`,
		},
		{
			name: "dot", prefix: "import . \"fmt\"\n",
			want: `Errorf(`, forbidden: `.Errorf(`,
		},
		{
			name: "collision", prefix: "var fmt, fmt_1 int\n",
			want: `fmt_2.Errorf`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			output := compileSourceOutput(t, "package sample\n"+test.prefix+body)
			if strings.Count(output, `"fmt"`) != 1 ||
				!strings.Contains(output, test.want) {
				t.Fatalf("generated output does not use %q\n%s", test.want, output)
			}
			if test.forbidden != "" && strings.Contains(output, test.forbidden) {
				t.Fatalf("generated output contains %q\n%s", test.forbidden, output)
			}
		})
	}
}

func TestPropagationVariableSpecificationsSeeEarlierNames(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func first() (int, error) { return 1, nil }
func second(value int) (int, error) { return value + 1, nil }

func values() (int, error) {
	var (
		firstValue = first()!!
		secondValue = second(firstValue)!!
	)
	return secondValue, nil
}
`)
	for _, required := range []string{
		"var firstValue, err = first()",
		"var secondValue, err_1 = second(firstValue)",
	} {
		if !strings.Contains(output, required) {
			t.Fatalf("generated output does not contain %q\n%s", required, output)
		}
	}
}

func TestPropagationVariableSpecificationsKeepOrder(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func mark(label string) int { return len(label) }
func load() (int, error) { return 1, nil }

func value() (int, error) {
	var (
		controlA = mark("control-a")
		controlB = mark("control-b")
	)
	var (
		before = mark("before")
		loaded = load()!!
		after = mark("after")
	)
	return controlA + controlB + before + loaded + after, nil
}
`)
	control := "var (\n\t\tcontrolA = mark(\"control-a\")\n" +
		"\t\tcontrolB = mark(\"control-b\")\n\t)"
	if !strings.Contains(output, control) {
		t.Fatalf("ordinary declaration changed\n%s", output)
	}
	ordered := []string{
		`var before = mark("before")`,
		"var loaded, err = load()",
		"if err != nil {",
		`var after = mark("after")`,
	}
	position := -1
	for _, required := range ordered {
		next := strings.Index(output, required)
		if next <= position {
			t.Fatalf("generated output puts %q out of order\n%s", required, output)
		}
		position = next
	}
}
