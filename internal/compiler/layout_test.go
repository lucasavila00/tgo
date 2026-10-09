package compiler

import (
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"testing"
)

func TestEnumAutoLayout(t *testing.T) {
	tests := []struct {
		name     string
		variants string
		boxed    []bool
		size     int64
	}{
		{"empty", "A struct{}; B struct{}", []bool{false, false}, 1},
		{"limit", "A struct{ Data [79]byte }", []bool{false}, 80},
		{"over", "A struct{ Data [80]byte }", []bool{true}, 24},
		{
			"largest", "A struct{ Data [40]byte }; B struct{ Data [60]byte }",
			[]bool{false, true}, 64,
		},
		{"tie", "A struct{ Data [40]byte }; B struct{ Data [40]byte }", []bool{true, false}, 64},
		{
			"repeat", "A struct{ Data [64]byte }; B struct{ Data [64]byte }; C struct{}",
			[]bool{true, true, false}, 24,
		},
		{"alignment", "A struct{ Data [9]int64 }; B struct{ Data byte }", []bool{true, false}, 24},
		{"zeroSize", "A struct{ Data [0]int64 }; B struct{}", []bool{false, false}, 16},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := layoutPackage(t, "package sample\ntype E enum {"+test.variants+"}\n")
			declaration := p.Models["E"]
			for i, variant := range declaration.Variants {
				if variant.Boxed != test.boxed[i] {
					t.Fatalf("%s boxed = %v, want %v", variant.Name, variant.Boxed, test.boxed[i])
				}
			}
			size := types.SizesFor("gc", "amd64").Sizeof(p.typed.Scope().Lookup("E").Type())
			if size != test.size {
				t.Fatalf("size = %d, want %d", size, test.size)
			}
		})
	}
}

func TestEnumAutoLayoutNested(t *testing.T) {
	p := layoutPackage(t, `package sample
 type Outer enum { A struct { Value Inner }; B struct { Data [40]byte } }
 type Inner enum { A struct { Data [200]byte } }
 `)
	if p.Models["Outer"].Variants[0].Boxed || p.Models["Outer"].Variants[1].Boxed {
		t.Fatal("outer payloads must fit inline after the inner payload is boxed")
	}
	if !p.Models["Inner"].Variants[0].Boxed {
		t.Fatal("inner payload must be boxed")
	}
}

func layoutPackage(t *testing.T, data string) *packageUnit {
	t.Helper()
	files := token.NewFileSet()
	parsed, err := parseSource(files, "sample.tgo", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	p := &packageUnit{
		Path: "sample", fs: files, Sources: []*source{parsed},
		Files: []*ast.File{parsed.File}, Models: make(map[string]*model),
		importer: importer.Default(),
	}
	for _, declaration := range parsed.Models {
		p.Models[declaration.Name] = declaration
	}
	outputs, err := p.compile()
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range outputs {
		if err := VerifyGeneratedModels("sample.tgo", []byte(data), output, p.typed); err != nil {
			t.Fatal(err)
		}
	}
	return p
}
