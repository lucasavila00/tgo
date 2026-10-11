package types_test

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

func TestPropagationTypes(t *testing.T) {
	for _, test := range []struct{ name, source, error string }{
		{"nested", `func call(v int) (int,error) {return v,nil}; func f() error {_=call(call(1)!)!!;return nil}`, ""},
		{"multiple", `func pair() (int,string,error) {return 1,"",nil}; func f() error {a,b:=pair()!;_,_=a,b;return nil}`, ""},
		{"error only", `func call() error {return nil}; func f() error {call()!;return nil}`, ""},
		{"generic", `func call[T any](v T) (T,error) {return v,nil}; func f() error {_=call(call(1)!)!;return nil}`, ""},
		{"missing error", `func call() int {return 1}; func f() error {_=call()!;return nil}`, "final error result"},
		{"missing return", `func call() error {return nil}; func f() {call()!}`, "final error interface return"},
		{"package scope", `func call() (int,error) {return 1,nil}; var v=call()!`, "final error interface return"},
		{"tuple count", `func call() (int,string,error) {return 1,"",nil}; func f() error {_=call()!;return nil}`, "assignment mismatch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			parsed, err := frontend.ParseFile(fset, "main.tgo", "main.tgo", []byte("package p\n"+test.source), make(map[string]bool))
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{TGo: make(map[string]types.TGoOperation)}
			for name, op := range parsed.Propagations {
				config.TGo[name] = types.TGoOperation{Wrap: op.Wrap, Position: op.Position}
			}
			_, err = config.Check("example.com/p", fset, []*ast.File{parsed.File}, nil)
			if test.error == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.error) {
				t.Fatalf("error %v; want %q", err, test.error)
			}
		})
	}
}
