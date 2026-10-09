package compiler

import (
	"go/importer"
	"go/token"
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

func NewPort(number int) (Port, error) {
	return Port{number: number}
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
	if count := strings.Count(output, ".check()"); count != 2 {
		t.Fatalf("generated check calls = %d, want 2\n%s", count, output)
	}
	if !strings.Contains(output, "return Port{}, invalid") {
		t.Fatal("the check method cannot return the invalid zero with an error")
	}
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
