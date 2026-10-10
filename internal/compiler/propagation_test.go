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

func TestPropagationGroupedVariableTracksEarlierResultType(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

type writer struct{}

func openWriter() (*writer, error) { return &writer{}, nil }
func (w *writer) Write() (int, error) { return 1, nil }

func use() (int, error) {
	var (
		w = openWriter()!!
		written = w.Write()!!
	)
	return written, nil
}
`)
	for _, required := range []string{
		"var w, err = openWriter()",
		"var written, err_1 = w.Write()",
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

func TestPropagationVariableSpecificationCommentsStayAttached(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func mark(label string) int { return len(label) }
func load() (int, error) { return 1, nil }

func value() (int, error) {
	// group doc
	var (
		// before doc
		before = mark("before") // before tail
		// loaded doc
		// loaded detail
		loaded = load()!! //nolint:errcheck // loaded tail
		// after doc
		after = mark("after") // after tail
	)
	return before + loaded + after, nil
}
`)
	want := `	// group doc

	// before doc
	var before = mark("before") // before tail

	// loaded doc
	// loaded detail
	var loaded, err = load() //nolint:errcheck // loaded tail
	if err != nil {
		return 0, err
	}
	// after doc
	var after = mark("after") // after tail`
	if !strings.Contains(output, want) {
		t.Fatalf("generated comments moved\n%s", output)
	}
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "sample.go", output, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&types.Config{}).Check(
		"sample", files, []*ast.File{file}, nil,
	); err != nil {
		t.Fatal(err)
	}
}

func TestPropagationNonDirectVariableCommentsStayAttached(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func mark(label string) int { return len(label) }
func load() (int, error) { return 1, nil }

func value() (int, error) {
	// group doc
	var (
		// typed doc
		typed int = load()!! //nolint:errcheck // typed inline
		// multiple doc
		first, second = mark("first"), load()!! // multiple inline
		// after doc
		after = mark("after") // after inline
	)
	return typed + first + second + after, nil
}
`)
	want := `	// group doc

	// typed doc
	result, err := load()
	if err != nil {
		return 0, err
	}
	var typed int = result //nolint:errcheck // typed inline

	// multiple doc
	operand := mark("first")
	result_1, err_1 := load()
	if err_1 != nil {
		return 0, err_1
	}
	var first, second = operand, result_1 // multiple inline

	// after doc
	var after = mark("after") // after inline`
	if !strings.Contains(output, want) {
		t.Fatalf("generated comments moved\n%s", output)
	}
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "sample.go", output, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&types.Config{}).Check(
		"sample", files, []*ast.File{file}, nil,
	); err != nil {
		t.Fatal(err)
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

func TestPropagationLowersForInitializer(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

type writer struct{}

func mark(value int) int { return value }
func openWriter(value int) (*writer, error) { return &writer{}, nil }
func (w *writer) ready() bool { return true }

func use() error {
	for before, w := mark(1), openWriter(2)!!; w.ready(); before++ {
		if before > 2 { break }
	}
	return nil
}
`)
	want := `{
		operand := mark(1)
		result, err := openWriter(2)
		if err != nil {
			return err
		}
		for before, w := operand, result; w.ready(); before++ {`
	if !strings.Contains(output, want) {
		t.Fatalf("generated output does not lower the for initializer\n%s", output)
	}
}

func TestPropagationKeepsLabeledForInitializerScope(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func load() (int, error) { return 1, nil }

func use() error {
	value := 0
	control := 0
	_ = control
outer:
	for value := load()!!; value < 2; value++ {
		continue outer
	}
	_ = value
	goto entry
entry:
	for other := load()!!; other < 2; other++ {
		break entry
	}
	return nil
}
`)
	for _, required := range []string{
		"result, err := load()",
		"outer:\n\t\tfor value := result; value < 2; value++",
		"entry:\n\t{\n\t\tresult_1, err_1 := load()",
		"control_1:\n\t\tfor other := result_1; other < 2; other++",
		"break control_1",
	} {
		if !strings.Contains(output, required) {
			t.Fatalf("generated output does not contain %q\n%s", required, output)
		}
	}
}

func TestPropagationForInitializerUsesFreshNames(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func load() (int, error) { return 1, nil }
func mark(value int) int { return value }

func use(result, err, operand, control int) (int, error) {
	for before, value := mark(operand), load()!!; value < 2; before++ {
		return result + err + before + control + value, nil
	}
	return 0, nil
}
`)
	for _, required := range []string{
		"operand_1 := mark(operand)",
		"result_1, err_1 := load()",
		"for before, value := operand_1, result_1; value < 2; before++",
	} {
		if !strings.Contains(output, required) {
			t.Fatalf("generated output does not contain %q\n%s", required, output)
		}
	}
}

func TestPropagationLowersTypeSwitchStatements(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func mark() int { return 1 }
func load() (any, error) { return "ready", nil }

func use() (string, error) {
	result, err := "", error(nil)
	switch prefix := mark(); value := (load()!!).(type) {
	case string:
		result = value
		_ = prefix
	}
	return result, err
}
`)
	want := `{
		prefix := mark()
		result_1, err_1 := load()
		if err_1 != nil {
			return "", err_1
		}
		switch value := (result_1).(type) {`
	if !strings.Contains(output, want) {
		t.Fatalf("generated output does not lower the type switch\n%s", output)
	}
}

func TestPropagationLowersEachTypeSwitchPart(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "bare guard",
			body: `switch (load()!!).(type) { case string: }`,
			want: []string{"result, err := load()", "switch (result).(type)"},
		},
		{
			name: "initializer",
			body: `switch prefix := text()!!; value := any("ready").(type) {
		case string: _, _ = prefix, value
	}`,
			want: []string{"prefix, err := text()", `switch value := any("ready").(type)`},
		},
		{
			name: "initializer and guard",
			body: `switch prefix := text()!!; value := (load()!!).(type) {
		case string: _, _ = prefix, value
	}`,
			want: []string{
				"prefix, err := text()",
				"result, err_1 := load()",
				"switch value := (result).(type)",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			output := compileSourceOutput(t, `package sample
func load() (any, error) { return "ready", nil }
func text() (string, error) { return "prefix", nil }
func use() error {
`+test.body+`
	return nil
}
`)
			position := -1
			for _, required := range test.want {
				next := strings.Index(output, required)
				if next <= position {
					t.Fatalf("generated output puts %q out of order\n%s", required, output)
				}
				position = next
			}
		})
	}
}

func TestPropagationLowersForPost(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func next(value int) (int, error) { return value + 1, nil }

func use(post, result int, err error) (int, error) {
	for value := 0; value < 2; value = next(value)!! {
		continue
	}
	return post + result, err
}
`)
	for _, required := range []string{
		"post_1 := false",
		"for value := 0; ; post_1 = true",
		"if post_1 {",
		"post_1 = false",
		"result_1, err_1 := next(value)",
		"value = result_1",
		"if !(value < 2)",
	} {
		if !strings.Contains(output, required) {
			t.Fatalf("generated output does not contain %q\n%s", required, output)
		}
	}
	if strings.Contains(output, "func()") {
		t.Fatalf("generated for post uses a function frame\n%s", output)
	}
}

func TestPropagationLowersForInitializerAndPost(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func first() (int, error) { return 0, nil }
func next(value int) (int, error) { return value + 1, nil }

func use() error {
	for value := first()!!; value < 2; value = next(value)!! {
	}
	return nil
}
`)
	ordered := []string{
		"result, err := first()",
		"post := false",
		"for value := result; ; post = true",
		"result_1, err_1 := next(value)",
		"value = result_1",
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

func TestPropagationLowersSwitchCasesInOrder(t *testing.T) {
	t.Parallel()
	output := compileSourceOutput(t, `package sample

func mark(name string, value int) int { return value }
func load(name string, value int) (int, error) { return value, nil }

func use(tag, selected, result int, err error) error {
outer:
	switch mark("tag", tag) {
	case mark("first", 1), load("second", 2)!!:
		fallthrough
	default:
		break outer
	case load("third", 3)!!:
	}
	return err
}
`)
	ordered := []string{
		`tag_1 := mark("tag", tag)`,
		"selected_1 := -1",
		`mark("first", 1)`,
		`result_1, err_1 := load("second", 2)`,
		"tag_1 == result_1",
		`result_2, err_2 := load("third", 3)`,
		"tag_1 == result_2",
		"outer:",
		"switch selected_1",
		"case 0:",
		"fallthrough",
		"default:",
		"break outer",
		"case 2:",
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
