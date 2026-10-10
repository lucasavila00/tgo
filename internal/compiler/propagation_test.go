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

func TestPropagationUsesEarlierPropagatedResultType(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample
type writer struct{}
func openWriter() (*writer, error) { return &writer{}, nil }
func (w *writer) Write() (int, error) { return 1, nil }
func use() error {
	w := openWriter()!!
	_ = w.Write()!!
	return nil
}
`)
	if !strings.Contains(output, "result, err_1 := w.Write()") ||
		!strings.Contains(output, "_ = result") {
		t.Fatalf("generated output does not lower the method call\n%s", output)
	}
}

func TestPropagationUsesTypeDerivedFromEarlierResult(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample
type writer struct{}
func openWriters() (map[string]*writer, error) { return nil, nil }
func (w *writer) Write() (int, error) { return 1, nil }
func use() error {
	writers := openWriters()!!
	w := writers["one"]
	_ = w.Write()!!
	return nil
}
`)
	if !strings.Contains(output, "result, err_1 := w.Write()") ||
		!strings.Contains(output, "_ = result") {
		t.Fatalf("generated output does not lower the method call\n%s", output)
	}
}

func TestPropagationTracksInferredResultScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "if initializer",
			body: `
func use() error {
	if w := openWriter()!!; w != nil {
		_ = w.Write()!!
	}
	return nil
}
`,
			want: "result, err_1 := w.Write()",
		},
		{
			name: "switch initializer",
			body: `
func use() error {
	switch w := openWriter()!!; {
	case w != nil:
		_ = w.Write()!!
	}
	return nil
}
`,
			want: "result, err_1 := w.Write()",
		},
		{
			name: "shadowed if initializer",
			body: `
type wideWriter struct{}
func openWideWriter() (*wideWriter, error) { return &wideWriter{}, nil }
func (w *wideWriter) Write() (int, int, error) { return 1, 2, nil }
func use() error {
	w := openWriter()!!
	if w := openWideWriter()!!; w != nil {
		_, _ = w.Write()!!
	}
	_ = w
	return nil
}
`,
			want: "result, result_1, err_2 := w.Write()",
		},
		{
			name: "var declaration",
			body: `
func use() error {
	var w = openWriter()!!
	_ = w.Write()!!
	return nil
}
`,
			want: "result, err_1 := w.Write()",
		},
		{
			name: "named map",
			body: `
type writers map[string]*writer
func openWriters() (writers, error) { return nil, nil }
func use() error {
	values := openWriters()!!
	w := values["one"]
	_ = w.Write()!!
	return nil
}
`,
			want: "result, err_1 := w.Write()",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			output := compileSourceOutput(t, `package sample
type writer struct{}
func openWriter() (*writer, error) { return &writer{}, nil }
func (w *writer) Write() (int, error) { return 1, nil }
`+test.body)
			if !strings.Contains(output, test.want) {
				t.Fatalf("generated output does not contain %q\n%s", test.want, output)
			}
		})
	}
}
