package compiler

import (
	"go/ast"
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
				fs:      files,
				Sources: []*source{parsed},
				Files:   []*ast.File{parsed.File},
				Models:  make(map[string]*model)}
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
	layoutPackage(t, `package sample
import "encoding/json"
import __tgo_json "fmt"
type E enum { A struct{} }
func use(value E) ([]byte,error) { __tgo_json.Println(value); return json.Marshal(value) }
`)
}

func TestVerifyEnumJSONMethods(t *testing.T) {
	data := []byte("package sample\ntype E enum { A struct{} }\n")
	files := token.NewFileSet()
	parsed, err := parseSource(files, "sample.tgo", data)
	if err != nil {
		t.Fatal(err)
	}
	p := &packageUnit{Path: "sample",
		fs:      files,
		Sources: []*source{parsed},
		Files:   []*ast.File{parsed.File},
		Models:  make(map[string]*model)}
	for _, model := range parsed.Models {
		p.Models[model.Name] = model
	}
	outputs, err := p.compile()
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range outputs {
		if err := VerifyGeneratedModels("sample.tgo", data, output, p.typed); err != nil {
			t.Fatal(err)
		}
		changed := strings.Replace(string(output), "MarshalJSON()", "MarshalChanged()", 1)
		if err := VerifyGeneratedModels("sample.tgo", data, []byte(changed), p.typed); err == nil {
			t.Fatal("changed JSON method was accepted")
		}
	}
}
