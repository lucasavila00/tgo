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
`)}},
		Imports: map[string]*CompiledPackage{"model": modelPackage},
		FileSet: token.NewFileSet(),
		Importer: packageImporter{
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
	_, problems = Compile(PackageInput{
		Path: "invalid",
		Sources: []File{{Name: "invalid.tgo", Data: []byte(`package invalid
import "model"
func Invalid(number int) (model.Port, error) { return model.NewPort(number) }
`)}},
		Imports: map[string]*CompiledPackage{"model": modelPackage},
		FileSet: token.NewFileSet(),
		Importer: packageImporter{
			packages: map[string]*types.Package{"model": modelPackage.Package},
			fallback: importer.Default(),
		},
	})
	if len(problems) == 0 ||
		!strings.Contains(problems[0].Error(), "NewPort is generated Go ABI") {
		t.Fatalf("error = %v, want imported generated Go ABI diagnostic", problems)
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
			want: "TgoPortInput is generated staging ABI",
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
			if len(problems) == 0 || !strings.Contains(problems[0].Error(), test.want) {
				t.Fatalf("error = %v, want %q", problems, test.want)
			}
		})
	}
}

func TestCheckedStructConstructorIsNotTGoAPI(t *testing.T) {
	t.Parallel()
	_, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample

type Port struct { number int } checked
func (value Port) check() (Port, error) { return value, nil }
func Invalid(number int) (Port, error) { return NewPort(number) }
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) == 0 ||
		!strings.Contains(problems[0].Error(), "NewPort is generated Go ABI") {
		t.Fatalf("error = %v, want generated Go ABI diagnostic", problems)
	}
}

func TestCheckedStructFieldsCannotBeChangedAfterConstruction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{name: "assignment", body: "value.number = 2"},
		{name: "increment", body: "value.number++"},
		{name: "address", body: "_ = &value.number"},
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
			if len(problems) == 0 ||
				!strings.Contains(problems[0].Error(), "checked field number cannot be changed") {
				t.Fatalf("error = %v, want checked field diagnostic", problems)
			}
		})
	}
}

func TestCheckedStructCheckCanOnlyNormalizeItsReceiver(t *testing.T) {
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
	if len(problems) == 0 ||
		!strings.Contains(problems[0].Error(), "checked field number cannot be changed") {
		t.Fatalf("error = %v, want non-receiver mutation diagnostic", problems)
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
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	return compiled
}

type packageImporter struct {
	packages map[string]*types.Package
	fallback types.Importer
}

func (i packageImporter) Import(path string) (*types.Package, error) {
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
