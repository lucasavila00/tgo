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
	port := model.Port{number: 1}!
	return model.Context{
		hidden: model.Hidden(1),
		alias: model.HiddenAlias(2),
		number: 255,
		port: port,
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
	if count := strings.Count(output, "model.NewPort("); count != 5 {
		t.Fatalf("generated constructor calls = %d, want 5\n%s", count, output)
	}
	if strings.Contains(output, ".check()") {
		t.Fatalf("imported construction calls private check\n%s", output)
	}
	if !strings.Contains(output, "model.TgoContextInput{") ||
		!strings.Contains(output, "FieldHidden: model.Hidden(1)") ||
		!strings.Contains(output, "FieldAlias: model.HiddenAlias(2)") ||
		!strings.Contains(output, "FieldNumber: 255") ||
		!strings.Contains(output, "FieldPort: port") {
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
		Path: "nested",
		Sources: []File{{Name: "nested.tgo", Data: []byte(`package nested
import "model"
type Context struct {
	port model.Port
	alias model.PortAlias
} checked
func (value Context) check() (Context, error) { return value, nil }
`)}},
		Imports: map[string]*CompiledPackage{"model": modelPackage},
		FileSet: token.NewFileSet(),
		Importer: checkedPackageImporter{
			packages: map[string]*types.Package{"model": modelPackage.Package},
			fallback: importer.Default(),
		},
	})
	if len(problems) != 0 {
		t.Fatalf("imported checked field failed: %v", problems)
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

func TestCheckedStructCheckLowersNestedLiteral(t *testing.T) {
	t.Parallel()
	compiled, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Inner struct { value int } checked
func (value Inner) check() (Inner, error) { return value, nil }

type Outer struct { inner Inner } checked
func (value Outer) check() (Outer, error) {
	replacement := Inner{value: 1}!
	value.inner = replacement
	return value, nil
}
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	if !strings.Contains(output, "return NewInner(tgoInput.FieldValue)") {
		t.Fatalf("nested literal in check skipped its constructor\n%s", output)
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

func TestCheckedStructFieldTypes(t *testing.T) {
	t.Parallel()
	_, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type namedBool bool
type namedInt int
type namedUint uint
type namedUintptr uintptr
type namedFloat float64
type namedComplex complex128
type namedString string
type boolAlias = namedBool
type intAlias = namedInt
type uintAlias = namedUint
type uintptrAlias = namedUintptr
type floatAlias = namedFloat
type complexAlias = namedComplex
type stringAlias = namedString

type inner struct { value int } checked
func (value inner) check() (inner, error) { return value, nil }
type innerAlias = inner

type embedded struct { value string } checked
func (value embedded) check() (embedded, error) { return value, nil }

type Value struct {
	boolean bool
	integer int
	unsigned uint
	uintptr uintptr
	byte byte
	rune rune
	floating float32
	complex complex64
	text string
	namedBoolean namedBool
	namedInteger namedInt
	namedUnsigned namedUint
	namedUintptr namedUintptr
	namedFloating namedFloat
	namedComplex namedComplex
	namedText namedString
	aliasBoolean boolAlias
	aliasInteger intAlias
	aliasUnsigned uintAlias
	aliasUintptr uintptrAlias
	aliasFloating floatAlias
	aliasComplex complexAlias
	aliasText stringAlias
	nested inner
	aliased innerAlias
	embedded
} checked

func (value Value) check() (Value, error) { return value, nil }
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
}

func TestCheckedStructRejectsUnsupportedFieldTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		declarations string
		fieldType    string
	}{
		{name: "pointer", fieldType: "*int"},
		{name: "pointer alias", declarations: "type bad = *int", fieldType: "bad"},
		{name: "non-nil pointer", fieldType: "%int"},
		{name: "non-nil pointer alias", declarations: "type bad = %int", fieldType: "bad"},
		{name: "slice", fieldType: "[]int"},
		{name: "slice alias", declarations: "type bad = []int", fieldType: "bad"},
		{name: "map", fieldType: "map[string]int"},
		{name: "map alias", declarations: "type bad = map[string]int", fieldType: "bad"},
		{name: "array", fieldType: "[1]int"},
		{name: "array alias", declarations: "type bad = [1]int", fieldType: "bad"},
		{name: "function", fieldType: "func()"},
		{name: "function alias", declarations: "type bad = func()", fieldType: "bad"},
		{name: "channel", fieldType: "chan int"},
		{name: "channel alias", declarations: "type bad = chan int", fieldType: "bad"},
		{name: "interface", fieldType: "any"},
		{name: "interface alias", declarations: "type bad = any", fieldType: "bad"},
		{
			name: "unsafe pointer", declarations: `import "unsafe"`,
			fieldType: "unsafe.Pointer",
		},
		{
			name: "unsafe pointer alias",
			declarations: `import "unsafe"
type bad = unsafe.Pointer`,
			fieldType: "bad",
		},
		{
			name: "enum", declarations: "type Choice enum { Ready struct{} }",
			fieldType: "Choice",
		},
		{
			name: "enum alias", declarations: `type Choice enum { Ready struct{} }
type bad = Choice`,
			fieldType: "bad",
		},
		{
			name: "ordinary struct", declarations: "type ordinary struct { value int }",
			fieldType: "ordinary",
		},
		{
			name: "ordinary struct alias",
			declarations: `type ordinary struct { value int }
type bad = ordinary`,
			fieldType: "bad",
		},
		{
			name: "defined checked type",
			declarations: `type Inner struct { value int } checked
func (value Inner) check() (Inner, error) { return value, nil }
type bad Inner`,
			fieldType: "bad",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, problems := Compile(PackageInput{
				Path: "sample",
				Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample
` + test.declarations + `
type Invalid struct { value ` + test.fieldType + ` } checked
func (value Invalid) check() (Invalid, error) { return value, nil }
`)}},
				FileSet: token.NewFileSet(), Importer: importer.Default(),
			})
			want := "checked struct field value must be boolean, numeric, string, " +
				"or a checked struct"
			if len(problems) == 0 || !strings.Contains(problems[0].Error(), want) {
				t.Fatalf("error = %v, want %q", problems, want)
			}
		})
	}
}

func compileCheckedStructPackage(t *testing.T) *CompiledPackage {
	t.Helper()
	compiled, problems := Compile(PackageInput{
		Path: "model",
		Sources: []File{{Name: "model.tgo", Data: []byte(`package model

type Port struct { number int } checked
func (value Port) check() (Port, error) { return value, nil }
type PortAlias = Port

type Hidden uint16
type HiddenAlias = Hidden

type Context struct {
	hidden Hidden
	alias HiddenAlias
	number uint8
	port Port
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
