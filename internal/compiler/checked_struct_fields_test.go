package compiler

import (
	"go/importer"
	"go/token"
	"strings"
	"testing"
)

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
