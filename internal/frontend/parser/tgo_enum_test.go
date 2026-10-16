package parser

import (
	stock "go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/internal/ast"
)

// TestEnumGrammar keeps declaration children and exhaustive clauses in the source tree.
func TestEnumGrammar(t *testing.T) {
	source := `package example
type Account enum {
	Personal struct { Name string }
	Business struct { Company string }
	Closed
}
func label(account Account) string {
	return switch v := account.(type) {
	case Account.Personal:
		v.Name
	case Account.Business:
		v.Company
	case Account.Closed:
		"closed"
	exhaustive /* coverage */:
	}
}`
	positions := token.NewFileSet()
	file, err := ParseTGoFile(positions, "account_tgo.go", "account.tgo", source, AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	var declaration *ast.EnumType
	var exhaustive *ast.CaseClause
	var names []string
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.EnumType:
			declaration = node
		case *ast.EnumVariant:
			names = append(names, node.Name.Name)
		case *ast.CaseClause:
			if node.Exhaustive {
				exhaustive = node
			}
		}
		return true
	})
	if declaration == nil || strings.Join(names, ",") != "Personal,Business,Closed" {
		t.Fatalf("enum declaration: %v; variants: %v", declaration, names)
	}
	if declaration.Variants[2].Type != nil || exhaustive == nil || len(exhaustive.Body) != 0 {
		t.Fatal("empty variant or exhaustive clause changed")
	}
	start := positions.Position(declaration.Pos()).Offset
	end := positions.Position(declaration.End()).Offset
	if source[start:start+4] != "enum" || source[end-1] != '}' {
		t.Fatalf("enum span: %d..%d", start, end)
	}
	if _, err := ParseFile(token.NewFileSet(), "account.go", source, AllErrors); err == nil {
		t.Fatal("ordinary Go parser accepted an enum declaration")
	}
}

// TestExhaustiveIdentifier keeps the clause word usable as an ordinary identifier.
func TestExhaustiveIdentifier(t *testing.T) {
	source := `package example
func f(x any) {
	switch x.(type) {
	case int:
		exhaustive := 1
		_ = exhaustive
	default:
	}
}`
	file, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", source, AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if clause, ok := node.(*ast.CaseClause); ok && clause.Exhaustive {
			t.Fatal("ordinary identifier became an exhaustive clause")
		}
		return true
	})
}

// TestParserSourceMetadata checks parser entry points independently of filenames and Go syntax.
func TestParserSourceMetadata(t *testing.T) {
	input := `package p
func f(value any) {
	switch value.(type) {
	case int:
	default:
	}
}`
	for _, test := range []struct {
		name     string
		filename string
		native   bool
	}{
		{"Go/go", "main.go", false},
		{"Go/tgo", "main.tgo", false},
		{"TGo/go", "main.go", true},
		{"TGo/tgo", "main.tgo", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			filename, native := test.filename, test.native
			mode := AllErrors | SkipObjectResolution
			var file *ast.File
			var err error
			if native {
				file, err = ParseTGoFile(token.NewFileSet(), filename, filename, input, mode)
			} else {
				file, err = ParseFile(token.NewFileSet(), filename, input, mode)
			}
			if err != nil {
				t.Fatal(err)
			}
			if file.TGo != native {
				t.Fatalf("file %s: TGo = %t, want %t", filename, file.TGo, native)
			}
			want, err := stock.ParseFile(token.NewFileSet(), filename, input, stock.Mode(mode))
			if err != nil || !sameGoTree(file, want) {
				t.Fatalf("ordinary Go tree differs: %v", err)
			}
			file.Name.Name = "changed"
			if sameGoTree(file, want) {
				t.Fatal("different package names have equal Go trees")
			}
		})
	}
}
