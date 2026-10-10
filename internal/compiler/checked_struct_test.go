package compiler

import (
	"go/importer"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestCheckedStructLiteralsCallCheck(t *testing.T) {
	t.Parallel()
	data := []byte(`package sample

var invalid error

type Port struct {
	number int
} checked

func (value Port) check() (Port, error) {
	if value.number < 1 {
		return Port{}, invalid
	}
	return value, nil
}

func MustPort(number int) (Port, error) {
	port := Port{number: number}!
	return port, nil
}
`)
	compiled, problems := Compile(PackageInput{
		Path: "sample", Sources: []File{{Name: "sample.tgo", Data: data}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	if strings.Contains(output, "} checked") {
		t.Fatal("generated Go contains the checked marker")
	}
	if count := strings.Count(output, ".check()"); count != 1 {
		t.Fatalf("generated check calls = %d, want 1\n%s", count, output)
	}
	if !strings.Contains(output, "func NewPort(number int)") ||
		!strings.Contains(output, "return NewPort(") {
		t.Fatalf("generated Go does not use NewPort\n%s", output)
	}
	if !strings.Contains(output, "return Port{}, invalid") {
		t.Fatal("the check method cannot return the invalid zero with an error")
	}
}

func TestCheckedStructConstructorAvoidsTypeNameParameter(t *testing.T) {
	t.Parallel()
	compiled, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type value struct { value int } checked

func (item value) check() (value, error) { return item, nil }

func Keyed(number int) (value, error) { return value{value: number} }
func Positional(number int) (value, error) { return value{number} }
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	for _, text := range []string{
		"func Newvalue(tgoField0 int) (value, error)",
		"return value{tgoField0}.check()",
		"return Newvalue(tgoInput.FieldValue)",
		"return Newvalue(number)",
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated Go does not contain %q\n%s", text, output)
		}
	}
	if count := strings.Count(output, ".check()"); count != 1 {
		t.Fatalf("generated check calls = %d, want 1\n%s", count, output)
	}
}

func TestImportedCheckedStructLiteralsUseConstructorABI(t *testing.T) {
	t.Parallel()
	modelPackage := compileCheckedStructPackage(t)
	compiled, problems := Compile(PackageInput{
		Path: "app",
		Sources: []File{{Name: "app.tgo", Data: []byte(`package app

import "model"

func Pair(number int) (model.Port, error) {
	return model.Port{number: number}
}

func Unkeyed(number int) (model.Port, error) {
	return model.Port{number}
}

func Wrapped(number int) (model.Port, error) {
	port := model.Port{number: number}!
	return port, nil
}

func Transparent(number int) (model.Port, error) {
	port := model.Port{number: number}!!
	return port, nil
}

func Contextual() (model.Context, error) {
	return model.Context{
		hidden: model.Hidden(1),
		points: []struct{ X int }{{X: 2}},
		number: 255,
		pointer: nil,
	}
}

func Unicode() (model.Unicode, error) {
	return model.Unicode{ς: 5, σ_1: 6, σ: 4}
}
`)}},
		Imports: map[string]*CompiledPackage{"model": modelPackage},
		FileSet: token.NewFileSet(),
		Importer: checkedPackageImporter{
			packages: map[string]*types.Package{"model": modelPackage.Package},
			fallback: importer.Default(),
		},
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["app.tgo"])
	if count := strings.Count(output, "model.NewPort("); count != 4 {
		t.Fatalf("generated constructor calls = %d, want 4\n%s", count, output)
	}
	if strings.Contains(output, ".check()") {
		t.Fatalf("imported construction calls private check\n%s", output)
	}
	if !strings.Contains(output, "model.TgoContextInput{") ||
		!strings.Contains(output, "FieldPoints: []struct{ X int }{{X: 2}}") ||
		!strings.Contains(output, "FieldHidden: model.Hidden(1)") ||
		!strings.Contains(output, "FieldNumber: 255") ||
		!strings.Contains(output, "FieldPointer: nil") {
		t.Fatalf("imported contextual construction changed\n%s", output)
	}
	for _, text := range []string{
		"model.TgoUnicodeInput{",
		"FieldΣ_1: 5",
		"FieldΣ_1_1: 6",
		"FieldΣ: 4",
		"model.NewUnicode(",
		".FieldΣ, ",
		".FieldΣ_1, ",
		".FieldΣ_1_1)",
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("imported Unicode construction does not contain %q\n%s", text, output)
		}
	}
	_, problems = Compile(PackageInput{
		Path: "invalid",
		Sources: []File{{Name: "invalid.tgo", Data: []byte(`package invalid
import "model"
func Invalid(number int) (model.Port, error) { return model.NewPort(number) }
`)}},
		Imports: map[string]*CompiledPackage{"model": modelPackage},
		FileSet: token.NewFileSet(),
		Importer: checkedPackageImporter{
			packages: map[string]*types.Package{"model": modelPackage.Package},
			fallback: importer.Default(),
		},
	})
	if len(problems) != 0 {
		t.Fatalf("compiler applied imported constructor policy: %v", problems)
	}
	_, problems = Compile(PackageInput{
		Path: "invalid",
		Sources: []File{{Name: "invalid.tgo", Data: []byte(`package invalid
import "model"
func Invalid(number int) (model.Port, error) {
	constructor := model.NewPort
	return constructor(number)
}
`)}},
		Imports: map[string]*CompiledPackage{"model": modelPackage},
		FileSet: token.NewFileSet(),
		Importer: checkedPackageImporter{
			packages: map[string]*types.Package{"model": modelPackage.Package},
			fallback: importer.Default(),
		},
	})
	if len(problems) != 0 {
		t.Fatalf("compiler applied imported constructor alias policy: %v", problems)
	}
}

func TestCheckedStructAliasLiteralsRewriteErasedImports(t *testing.T) {
	t.Parallel()
	modelPackage := compileCheckedStructPackage(t)
	bridgePackage, problems := Compile(PackageInput{
		Path: "bridge",
		Sources: []File{{Name: "bridge.tgo", Data: []byte(`package bridge
import "model"
type Alias = model.Port
func Marker() int { return 1 }
`)}},
		Imports: map[string]*CompiledPackage{"model": modelPackage},
		FileSet: token.NewFileSet(),
		Importer: checkedPackageImporter{
			packages: map[string]*types.Package{"model": modelPackage.Package},
			fallback: importer.Default(),
		},
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}

	tests := []struct {
		name        string
		imports     string
		body        string
		wants       []string
		forbidden   string
		constructor string
	}{
		{
			name: "literal only",
			imports: `import (
	"bridge"
	_ "unsafe"
)`,
			body: `func Make(number int) (any, error) {
	return bridge.Alias{number: number}
}`,
			wants:       []string{`_ "bridge"`},
			forbidden:   "\n\t\"bridge\"\n",
			constructor: "model.NewPort(",
		},
		{
			name: "named literal only",
			imports: `import (
	records "model"
	facade "bridge"
)`,
			body: `func Make(number int) (records.Port, error) {
	return facade.Alias{number: number}
}`,
			wants:       []string{`_ "bridge"`},
			forbidden:   `facade "bridge"`,
			constructor: "records.NewPort(",
		},
		{
			name: "other source use",
			imports: `import (
	"model"
	facade "bridge"
)`,
			body: `func Make(number int) (model.Port, error) { return facade.Alias{number: number} }
func Marker() int { return facade.Marker() }`,
			wants:       []string{`facade "bridge"`},
			forbidden:   `_ "bridge"`,
			constructor: "model.NewPort(",
		},
		{
			name: "duplicate import path",
			imports: `import (
	"model"
	used "bridge"
	erased "bridge"
)`,
			body: `func Make(number int) (model.Port, error) {
	return erased.Alias{number: number + used.Marker()}
}`,
			wants:       []string{`used "bridge"`, `_ "bridge"`},
			forbidden:   `erased "bridge"`,
			constructor: "model.NewPort(",
		},
		{
			name: "dot literal only",
			imports: `import (
	"model"
	. "bridge"
	used "bridge"
)`,
			body: `func Make(number int) (model.Port, error) {
	return Alias{number: number + used.Marker()}
}`,
			wants:       []string{`used "bridge"`, `_ "bridge"`},
			forbidden:   `. "bridge"`,
			constructor: "model.NewPort(",
		},
		{
			name: "dot other source use",
			imports: `import (
	"model"
	. "bridge"
	used "bridge"
)`,
			body: `func Make(number int) (model.Port, error) {
	return Alias{number: number + used.Marker()}
}
func ReadMarker() int { return Marker() }`,
			wants:       []string{`used "bridge"`, `. "bridge"`},
			forbidden:   `_ "bridge"`,
			constructor: "model.NewPort(",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			compiled, problems := Compile(PackageInput{
				Path: "app",
				Sources: []File{{
					Name: "app.tgo",
					Data: []byte(
						"package app\n" + test.imports + "\n" + test.body + "\n",
					),
				}},
				Imports: map[string]*CompiledPackage{
					"bridge": bridgePackage,
					"model":  modelPackage,
				},
				FileSet: token.NewFileSet(),
				Importer: checkedPackageImporter{
					packages: map[string]*types.Package{
						"bridge": bridgePackage.Package,
						"model":  modelPackage.Package,
					},
					fallback: importer.Default(),
				},
			})
			if len(problems) != 0 {
				t.Fatal(problems[0])
			}
			output := string(compiled.Outputs["app.tgo"])
			for _, want := range test.wants {
				if !strings.Contains(output, want) {
					t.Fatalf("generated imports do not contain %q\n%s", want, output)
				}
			}
			if strings.Contains(output, test.forbidden) {
				t.Fatalf("generated imports contain %q\n%s", test.forbidden, output)
			}
			if !strings.Contains(output, test.constructor) {
				t.Fatalf("generated output does not use the defining constructor\n%s", output)
			}
			if strings.Contains(test.imports, `_ "unsafe"`) &&
				!strings.Contains(output, `_ "unsafe"`) {
				t.Fatalf("generated output removed a source blank import\n%s", output)
			}
		})
	}
}

func TestLocalCheckedStructAliasLiteralUsesConstructor(t *testing.T) {
	t.Parallel()
	compiled, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Port struct { number int } checked
func (value Port) check() (Port, error) { return value, nil }
type Alias = Port
func Make(number int) (Port, error) { return Alias{number: number} }
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	if !strings.Contains(output, "NewPort(tgoInput.FieldNumber)") {
		t.Fatalf("local alias did not use the checked constructor\n%s", output)
	}
}

func TestCheckedStructLiteralPreservesEvaluationOrder(t *testing.T) {
	t.Parallel()
	compiled, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Record struct {
	first int
	second int = observe(2)
	third int
} checked

func (value Record) check() (Record, error) { return value, nil }
func observe(value int) int { return value }

func Make() (Record, error) {
	return Record{third: observe(3), first: observe(1), ..default}
}
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	third := strings.Index(output, "FieldThird: observe(3)")
	first := strings.Index(output, "FieldFirst: observe(1)")
	second := strings.Index(output, "FieldSecond: TgoDefaultRecordsecond()")
	call := strings.Index(output, "return NewRecord(tgoInput.FieldFirst")
	if third < 0 || first < third || second < first || call < 0 {
		t.Fatalf("checked literal evaluation order changed\n%s", output)
	}
	if strings.Count(output, "observe(3)") != 1 ||
		strings.Count(output, "observe(1)") != 1 ||
		strings.Count(output, "FieldSecond: TgoDefaultRecordsecond()") != 1 {
		t.Fatalf("checked literal expression is not evaluated once\n%s", output)
	}
}

func TestCheckedStructCarrierFieldNamesAreUnique(t *testing.T) {
	t.Parallel()
	compiled, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Pair struct {
	σ int
	ς int
	σ_1 int
} checked

func (value Pair) check() (Pair, error) { return value, nil }

func Make() (Pair, error) {
	return Pair{ς: 2, σ_1: 3, σ: 1}
}
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	for _, text := range []string{
		"FieldΣ_1: 2",
		"FieldΣ_1_1: 3",
		"FieldΣ: 1",
		"return NewPair(tgoInput.FieldΣ, tgoInput.FieldΣ_1, tgoInput.FieldΣ_1_1)",
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated Go does not contain %q\n%s", text, output)
		}
	}
}

func TestCheckedStructCarrierIsGeneratedOnly(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "source reference",
			body: "var invalid TgoPortInput",
			want: "",
		},
		{
			name: "name collision",
			body: "type TgoPortInput struct{}",
			want: "TgoPortInput redeclared",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, problems := Compile(PackageInput{
				Path: "sample",
				Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Port struct { number int } checked
func (value Port) check() (Port, error) { return value, nil }
` + test.body + "\n")}},
				FileSet: token.NewFileSet(), Importer: importer.Default(),
			})
			if test.want == "" && len(problems) != 0 {
				t.Fatalf("compiler applied staging ABI policy: %v", problems)
			}
			if test.want != "" &&
				(len(problems) == 0 || !strings.Contains(problems[0].Error(), test.want)) {
				t.Fatalf("error = %v, want %q", problems, test.want)
			}
		})
	}
}

func TestCompilerLeavesCheckedConstructorPolicyToTgolint(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"return NewPort(number)",
		"constructor := NewPort; return constructor(number)",
	} {
		_, problems := Compile(PackageInput{
			Path: "sample",
			Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Port struct { number int } checked
func (value Port) check() (Port, error) { return value, nil }
func Invalid(number int) (Port, error) { ` + body + ` }
`)}},
			FileSet: token.NewFileSet(), Importer: importer.Default(),
		})
		if len(problems) != 0 {
			t.Fatalf("compiler applied generated ABI policy: %v", problems)
		}
	}
}

func TestCompilerLeavesCheckedFieldPolicyToTgolint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{name: "assignment", body: "value.number = 2"},
		{name: "increment", body: "value.number++"},
		{name: "address", body: "_ = &value.number"},
		{name: "range assignment", body: "for value.number = range []int{1} {}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, problems := Compile(PackageInput{
				Path: "sample",
				Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Port struct { number int } checked
func (value Port) check() (Port, error) { value.number = 1; return value, nil }
func Invalid(value Port) { ` + test.body + ` }
`)}},
				FileSet: token.NewFileSet(), Importer: importer.Default(),
			})
			if len(problems) != 0 {
				t.Fatalf("compiler applied checked field policy: %v", problems)
			}
		})
	}
}

func TestCompilerLeavesCheckedReceiverPolicyToTgolint(t *testing.T) {
	t.Parallel()
	_, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Port struct { number int } checked
func (value Port) check() (Port, error) {
	value.number = 1
	other := value
	other.number = 2
	return value, nil
}
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compiler applied checked receiver policy: %v", problems)
	}
}

func TestCheckedStructBlankFieldUsesUnkeyedConstructor(t *testing.T) {
	t.Parallel()
	compiled, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Padded struct { _ int; value int } checked
func (value Padded) check() (Padded, error) { return value, nil }
func Make(value int) (Padded, error) { return Padded{0, value} }
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	if !strings.Contains(output, "func NewPadded(tgoField0 int, value int)") ||
		!strings.Contains(output, "return NewPadded(0, value)") {
		t.Fatalf("blank field constructor changed unkeyed literals\n%s", output)
	}
}

func compileCheckedStructPackage(t *testing.T) *CompiledPackage {
	t.Helper()
	compiled, problems := Compile(PackageInput{
		Path: "model",
		Sources: []File{{Name: "model.tgo", Data: []byte(`package model

type Port struct { number int } checked
func (value Port) check() (Port, error) { return value, nil }

type hidden struct { X int }
func Hidden(value int) hidden { return hidden{X: value} }

type Context struct {
	pointer *int
	number uint8
	hidden hidden
	points []struct { X int }
} checked
func (value Context) check() (Context, error) { return value, nil }

type Unicode struct {
	σ int
	ς int
	σ_1 int
} checked
func (value Unicode) check() (Unicode, error) { return value, nil }
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	return compiled
}

type checkedPackageImporter struct {
	packages map[string]*types.Package
	fallback types.Importer
}

func (i checkedPackageImporter) Import(path string) (*types.Package, error) {
	if imported := i.packages[path]; imported != nil {
		return imported, nil
	}
	return i.fallback.Import(path)
}

func TestCheckedStructDeclarationErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		declaration string
		want        string
	}{
		{
			name: "public field",
			declaration: `type Port struct { Number int } checked
func (value Port) check() (Port, error) { return value, nil }
`,
			want: "checked struct field Number must be private",
		},
		{
			name: "missing method",
			declaration: `type Port struct { number int } checked
`,
			want: "checked struct Port needs check() (Port, error)",
		},
		{
			name: "pointer receiver",
			declaration: `type Port struct { number int } checked
func (value *Port) check() (Port, error) { return *value, nil }
`,
			want: "check method must have signature check() (Port, error)",
		},
		{
			name: "wrong result",
			declaration: `type Port struct { number int } checked
func (value Port) check() (Port, bool) { return value, true }
`,
			want: "check method must have signature check() (Port, error)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, problems := Compile(PackageInput{
				Path: "sample",
				Sources: []File{{
					Name: "sample.tgo",
					Data: []byte("package sample\n" + test.declaration),
				}},
				FileSet: token.NewFileSet(), Importer: importer.Default(),
			})
			if len(problems) == 0 || !strings.Contains(problems[0].Error(), test.want) {
				t.Fatalf("error = %v, want %q", problems, test.want)
			}
		})
	}
}
