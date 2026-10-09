package compiler

import (
	"go/importer"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestImportedDefaultsUseReadableOwnerImports(t *testing.T) {
	t.Parallel()
	model, problems := Compile(PackageInput{
		Path: "example.com/model",
		Sources: []File{{Name: "model.tgo", Data: []byte(`package model
type Record struct { Value int = 7 }
`)}},
		FileSet: token.NewFileSet(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	aliasFile := File{Name: "alias.go", Data: []byte(`package sample
import records "example.com/model"
type Record = records.Record
`)}
	tests := []struct {
		name      string
		source    string
		goFiles   []File
		want      string
		forbidden string
	}{
		{
			name: "normal", source: `package sample
import "example.com/model"
func makeRecord() model.Record { return model.Record{..default} }
`, want: `model.TgoDefaultRecordValue()`,
		},
		{
			name: "custom", source: `package sample
import records "example.com/model"
func makeRecord() records.Record { return records.Record{..default} }
`, want: `records.TgoDefaultRecordValue()`,
		},
		{
			name: "blank", source: `package sample
import _ "example.com/model"
func makeRecord() Record { return Record{..default} }
`, goFiles: []File{aliasFile}, want: `model.TgoDefaultRecordValue()`,
			forbidden: `import _ "example.com/model"`,
		},
		{
			name: "dot", source: `package sample
import . "example.com/model"
func makeRecord() Record { return Record{..default} }
`, want: `TgoDefaultRecordValue()`,
		},
		{
			name: "collision", source: `package sample
var model, model_1 int
func makeRecord() Record { return Record{..default} }
`, goFiles: []File{aliasFile}, want: `model_2.TgoDefaultRecordValue()`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			compiled, problems := Compile(PackageInput{
				Path:    "sample",
				Sources: []File{{Name: "sample.tgo", Data: []byte(test.source)}},
				GoFiles: test.goFiles,
				Imports: map[string]*CompiledPackage{"example.com/model": model},
				FileSet: token.NewFileSet(),
				Importer: packageImporter{
					"example.com/model": model.Package,
				},
			})
			if len(problems) != 0 {
				t.Fatal(problems[0])
			}
			output := string(compiled.Outputs["sample.tgo"])
			if strings.Count(output, `"example.com/model"`) != 1 ||
				!strings.Contains(output, test.want) {
				t.Fatalf("generated output does not contain %q\n%s", test.want, output)
			}
			if test.forbidden != "" && strings.Contains(output, test.forbidden) {
				t.Fatalf("generated output contains %q\n%s", test.forbidden, output)
			}
		})
	}
}

func TestImportedModelLoweringUsesReadableOwnerImport(t *testing.T) {
	t.Parallel()
	model, problems := Compile(PackageInput{
		Path: "example.com/model",
		Sources: []File{{Name: "model.tgo", Data: []byte(`package model
type Account enum { Personal struct { Name string } }
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	bridge, problems := Compile(PackageInput{
		Path: "example.com/bridge",
		GoFiles: []File{{Name: "bridge.go", Data: []byte(`package bridge
import "example.com/model"
type Account = model.Account
`)}},
		FileSet: token.NewFileSet(),
		Importer: packageImporter{
			"example.com/model": model.Package,
		},
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	compiled, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "sample.tgo", Data: []byte(`package sample
import "example.com/bridge"
var model, model_1 int
func makeAccount() any {
	return bridge.Account.Personal{Name: "name"}
}
`)}},
		Imports: map[string]*CompiledPackage{
			"example.com/bridge": bridge,
			"example.com/model":  model,
		},
		FileSet: token.NewFileSet(),
		Importer: packageImporter{
			"example.com/bridge": bridge.Package,
			"example.com/model":  model.Package,
		},
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	for _, want := range []string{
		`model_2 "example.com/model"`,
		`model_2.NewAccountPersonal(input.FieldName)`,
		`model_2.TgoAccountPersonalInput{FieldName: "name"}`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("generated output does not contain %q\n%s", want, output)
		}
	}
}

type packageImporter map[string]*types.Package

func (i packageImporter) Import(path string) (*types.Package, error) {
	if imported := i[path]; imported != nil {
		return imported, nil
	}
	return importer.Default().Import(path)
}
