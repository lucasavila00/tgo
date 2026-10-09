package compiler

import (
	"go/ast"
	"go/importer"
	"go/token"
	"strings"
	"testing"
)

func TestEnumJSONControls(t *testing.T) {
	tests := []struct{ name, declaration, errorText string }{
		{"default", "type E enum { A struct{} }", ""},
		{"external", "type E enum `json:\"external\"` { A struct{} }", ""},
		{"internal", "type E enum `json:\"internal,tag=type\"` { A struct{} }", ""},
		{"adjacent", "type E enum `json:\"adjacent,tag=type,content=data\"` { A struct{} }", ""},
		{"untagged", "type E enum `json:\"untagged\"` { A struct{} }", ""},
		{"unknown form", "type E enum `json:\"other\"` { A struct{} }", "unknown enum JSON form"},
		{"empty form", "type E enum `json:\"\"` { A struct{} }", "unknown enum JSON form"},
		{"unknown option",
			"type E enum `json:\"external,other=x\"` { A struct{} }",
			"unknown enum JSON option"},
		{"empty option",
			"type E enum `json:\"internal,tag=\"` { A struct{} }",
			"invalid enum JSON option"},
		{"bare option",
			"type E enum `json:\"internal,tag\"` { A struct{} }",
			"invalid enum JSON option"},
		{"duplicate option",
			"type E enum `json:\"internal,tag=a,tag=b\"` { A struct{} }",
			"invalid enum JSON option"},
		{"external tag",
			"type E enum `json:\"external,tag=type\"` { A struct{} }",
			"does not use tag or content"},
		{"untagged content",
			"type E enum `json:\"untagged,content=data\"` { A struct{} }",
			"does not use tag or content"},
		{"internal missing tag",
			"type E enum `json:\"internal\"` { A struct{} }",
			"internal JSON requires tag"},
		{"internal content",
			"type E enum `json:\"internal,tag=type,content=data\"` { A struct{} }",
			"internal JSON requires tag"},
		{"adjacent missing tag",
			"type E enum `json:\"adjacent,content=data\"` { A struct{} }",
			"adjacent JSON requires"},
		{"adjacent missing content",
			"type E enum `json:\"adjacent,tag=type\"` { A struct{} }",
			"adjacent JSON requires"},
		{"same keys",
			"type E enum `json:\"adjacent,tag=type,content=type\"` { A struct{} }",
			"different tag and content"},
		{"empty variant", "type E enum { A struct{} `json:\"\"` }", "empty or duplicate"},
		{"duplicate variant",
			"type E enum { A struct{} `json:\"same\"`; B struct{} `json:\"same\"` }",
			"empty or duplicate"},
		{"default name conflict",
			"type E enum { A struct{} `json:\"B\"`; B struct{} }",
			"empty or duplicate"},
		{"other tag", "type E enum `xml:\"E\"` { A struct{} `xml:\"a\"` }", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_,
				err := parseSource(token.NewFileSet(),
				"sample.tgo",
				[]byte("package sample\n"+test.declaration+"\n"))
			if test.errorText == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.errorText) {
				t.Fatalf("error = %v, want %q", err, test.errorText)
			}
		})
	}
}

func TestEnumJSONInternalFields(t *testing.T) {
	tests := []struct {
		name, fields, other string
		conflict            bool
	}{
		{"default name", "Type string", "", true},
		{"folded name before", "Name string `json:\"type\"`; ID string", "", true},
		{"folded name after", "ID string; Name string `json:\"type\"`", "", true},
		{"folded uppercase name", "Name string `json:\"TYPE\"`", "", true},
		{"renamed field", "Name string `json:\"Type\"`", "", true},
		{"omitted name", "Type string `json:\",omitempty\"`", "", true},
		{"ignored field", "Type string `json:\"-\"`", "", false},
		{"renamed away", "Type string `json:\"value\"`", "", false},
		{"unexported field", "hidden string `json:\"Type\"`", "", false},
		{"embedded", "Embedded", "type Embedded struct { Value string `json:\"Type\"` }", true},
		{"embedded pointer", "*Embedded", "type Embedded struct { Type string }", true},
		{"named embedded",
			"Embedded `json:\"Type\"`",
			"type Embedded struct { Value string }",
			true},
		{"embedded renamed",
			"Embedded `json:\"value\"`",
			"type Embedded struct { Type string }",
			false},
		{"embedded ignored",
			"Embedded `json:\"-\"`",
			"type Embedded struct { Type string }",
			false},
		{"hidden embedded", "embedded", "type embedded struct { Type string }", true},
		{"ambiguous embedded",
			"Left; Right",
			"type Left struct { Type string }; type Right struct { Type string }",
			false},
		{"tag dominates",
			"Left; Right",
			"type Left struct { Type string }; type Right struct { Value string `json:\"Type\"` }",
			true},
		{"recursive embedded",
			"*Embedded",
			"type Embedded struct { *Embedded; Value string }",
			false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := "package sample\n" +
				test.other +
				"\ntype E enum `json:\"internal,tag=Type\"` { A struct {" +
				test.fields +
				"} }\n"
			files := token.NewFileSet()
			parsed, err := parseSource(files, "sample.tgo", []byte(data))
			if err != nil {
				t.Fatal(err)
			}
			p := &packageUnit{Path: "sample",
				fs:       files,
				Sources:  []*source{parsed},
				Files:    []*ast.File{parsed.File},
				Models:   make(map[string]*model),
				importer: importer.Default()}
			for _, model := range parsed.Models {
				p.Models[model.Name] = model
			}
			_, err = p.compile()
			if test.conflict {
				if err == nil || !strings.Contains(err.Error(), "conflicts with payload field") {
					t.Fatalf("error = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEnumJSONImportNames(t *testing.T) {
	output := compileSourceOutput(t, `package sample
import json "math"
import jsonv2 "bytes"
import jsontext "io"
import fmt "errors"
import strings "strconv"
type E enum `+"`json:\"adjacent,tag=type,content=data\"`"+` {
	A struct { Name string `+"`json:\"name\"`"+` }
}
type F enum `+"`json:\"external\"`"+` { B struct{} }
func tgoEExternalJSONTo() {}
func tgoEAdjacentJSONTo() {}
func use(value E) {
	_ = json.Abs(1)
	_ = jsonv2.Compare
	_ = jsontext.EOF
	_ = fmt.New
	_ = strings.Itoa
}
`)
	for _, name := range []string{
		"json_1", "jsonv2_1", "jsontext_1", "fmt_1", "strings_1",
		"tgoEExternalJSONTo_1", "tgoEAdjacentJSONTo_1",
	} {
		if !strings.Contains(output, name) {
			t.Fatalf("generated output does not contain fallback %s\n%s", name, output)
		}
	}
}

func TestEnumJSONReusesCompatibleImports(t *testing.T) {
	output := compileSourceOutput(t, `package sample
import standardjson "encoding/json"
import "fmt"
type E enum { A struct{} }
`)
	if strings.Count(output, `"encoding/json"`) != 1 ||
		strings.Count(output, `"fmt"`) != 1 ||
		!strings.Contains(output, "standardjson.Marshal") ||
		!strings.Contains(output, "fmt.Sprintf") {
		t.Fatalf("generated output did not reuse compatible imports\n%s", output)
	}
}

func TestEnumJSONRewritesBlankImport(t *testing.T) {
	output := compileSourceOutput(t, `package sample
import _ "encoding/json"
type E enum { A struct{} }
`)
	if strings.Count(output, `"encoding/json"`) != 1 ||
		!strings.Contains(output, `import "encoding/json"`) ||
		!strings.Contains(output, "json.Marshal") {
		t.Fatalf("generated output did not reuse the blank import\n%s", output)
	}
}

func TestEnumJSONReusesDotImport(t *testing.T) {
	output := compileSourceOutput(t, `package sample
import . "encoding/json"
type E enum { A struct{} }
`)
	if strings.Count(output, `"encoding/json"`) != 1 ||
		!strings.Contains(output, `. "encoding/json"`) ||
		!strings.Contains(output, "return Marshal(") ||
		strings.Contains(output, "tgoDotImport") {
		t.Fatalf("generated output did not reuse the dot import\n%s", output)
	}
}

func TestGeneratedNamesAreReadable(t *testing.T) {
	output := compileSourceOutput(t, `package sample
type E enum { A struct{} }
func values() []int { return []int{1} }
func read(result int, index int) []int {
	source := 1
	_ = source
	return []int{for _, value := range values() { value + 1 }}
}
`)
	for _, text := range []string{
		`"encoding/json"`,
		`jsonv2 "encoding/json/v2"`,
		`"encoding/json/jsontext"`,
		`"fmt"`,
		"func tgoEExternalJSONTo",
		"result_1",
		"index_1",
		"source_1",
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated output does not contain %q\n%s", text, output)
		}
	}
	if strings.Contains(output, "__tgo_") {
		t.Fatalf("generated output contains an old synthetic name\n%s", output)
	}
}

func TestEnumJSONHelperNamesAvoidPackageDeclarations(t *testing.T) {
	t.Parallel()
	enumSource := File{Name: "enum.tgo", Data: []byte(`package sample
type E enum ` + "`json:\"external\"`" + ` { A struct{} }
type F enum ` + "`json:\"adjacent,tag=type,content=data\"`" + ` { B struct{} }
func useNames() {
	var tgoEExternalJSONTo_2, tgoEAdjacentJSONTo_2 int
	_, _ = tgoEExternalJSONTo_2, tgoEAdjacentJSONTo_2
}
`)}
	declarations := []byte(`package sample
func tgoEExternalJSONTo() {}
func tgoEExternalJSONTo_1() {}
var tgoEAdjacentJSONTo, tgoEAdjacentJSONTo_1 func()
`)
	tests := []struct {
		name    string
		sources []File
		goFiles []File
	}{
		{
			name: "tgo declarations",
			sources: []File{
				enumSource,
				{Name: "names.tgo", Data: declarations},
			},
		},
		{
			name:    "go declarations",
			sources: []File{enumSource},
			goFiles: []File{{Name: "names.go", Data: declarations}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			compiled, problems := Compile(PackageInput{
				Path: "sample", Sources: test.sources, GoFiles: test.goFiles,
				FileSet: token.NewFileSet(), Importer: importer.Default(),
			})
			if len(problems) != 0 {
				t.Fatal(problems[0])
			}
			output := string(compiled.Outputs["enum.tgo"])
			for _, name := range []string{
				"tgoEExternalJSONTo_3", "tgoEAdjacentJSONTo_3",
			} {
				if !strings.Contains(output, name) {
					t.Fatalf("generated output does not contain %s\n%s", name, output)
				}
			}
		})
	}
}

func compileSourceOutput(t *testing.T, source string) string {
	t.Helper()
	compiled, problems := Compile(PackageInput{
		Path: "sample", Sources: []File{{Name: "sample.tgo", Data: []byte(source)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	return string(compiled.Outputs["sample.tgo"])
}

func TestEnumJSONMethodsGenerated(t *testing.T) {
	data := []byte("package sample\ntype E enum { A struct{} }\n")
	files := token.NewFileSet()
	parsed, err := parseSource(files, "sample.tgo", data)
	if err != nil {
		t.Fatal(err)
	}
	p := &packageUnit{Path: "sample",
		fs:       files,
		Sources:  []*source{parsed},
		Files:    []*ast.File{parsed.File},
		Models:   make(map[string]*model),
		importer: importer.Default()}
	for _, model := range parsed.Models {
		p.Models[model.Name] = model
	}
	outputs, err := p.compile()
	if err != nil {
		t.Fatal(err)
	}
	output := string(outputs["sample.tgo"])
	for _, method := range []string{
		"MarshalJSON()",
		"MarshalJSONTo(",
		"UnmarshalJSON(",
		"UnmarshalJSONFrom(",
	} {
		if !strings.Contains(output, method) {
			t.Fatalf("generated output does not contain %s", method)
		}
	}
}
