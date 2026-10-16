package types_test

import (
	"go/token"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend/parser"
)

// TestEnumGoSourceRules keeps ordinary Go API access separate from native enum syntax.
func TestEnumGoSourceRules(t *testing.T) {
	for _, test := range []struct{ name, source, diagnostic string }{
		{"storage", `var a = Account{tgoTag:99}; var b = Account(Raw{tgoTag:99})`, ""},
		{"generic", `func forge[T ~Raw]() T { return T{tgoTag:99} }; var a = forge[Account]()`, ""},
		{"public_api", `var a = NewAccountPersonal("name"); var b = a.PersonalPayload().Name`, ""},
		{"native_constructor", `var a = Account.Personal{Name:"name"}`, "Personal"},
		{"native_switch", `func f(a Account) { switch a.(type) { default: } }`, "not an interface"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			native, err := parser.ParseTGoFile(fset, "enum.input", "enum.input", "package p\n"+enumGenericSource, 0)
			if err != nil {
				t.Fatal(err)
			}
			ordinary, err := parser.ParseFile(fset, "caller.go", "package p\n"+test.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{}
			_, err = config.Check("example.com/p", fset, []*ast.File{native, ordinary}, &types.Info{})
			if test.diagnostic == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.diagnostic) {
				t.Fatalf("want %q, got %v", test.diagnostic, err)
			}
		})
	}
}
