package types_test

import (
	"go/token"
	"testing"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend/parser"
)

// TestEnumInvalidCyclesWithErrors stops layout after cycle errors even when checking continues.
func TestEnumInvalidCyclesWithErrors(t *testing.T) {
	for _, source := range []string{
		`type A enum { First struct { Value A } }`,
		`type A enum { First struct { Value B } }; type B enum { First struct { Value A } }`,
		`type A enum { First struct { Value B } }; type B struct { Value A }`,
		`type A enum { First struct { Value [1]A } }`,
	} {
		t.Run(source, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseTGoFile(fset, "input.tgo", "input.tgo", "package p\n"+source, 0)
			if err != nil {
				t.Fatal(err)
			}
			var errors []error
			config := types.Config{Error: func(err error) { errors = append(errors, err) }}
			_, err = config.Check("example.com/p", fset, []*ast.File{file}, &types.Info{})
			if err == nil || len(errors) == 0 {
				t.Fatal("recursive value type was accepted")
			}
		})
	}
}
