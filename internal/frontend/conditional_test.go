package frontend

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

func TestParseConditionalExpressions(t *testing.T) {
	for _, source := range []string{
		`func f(ok bool) string { return if ok { "yes" } else { "no" } }`,
		`func f(ok bool) { x := if ok { log(); "yes" } else { "no" }; use(x) }`,
		`func f(ok bool) { use(if ok { 1 } else { 2 }, if !ok { 3 } else { 4 }) }`,
		`func f(ok bool) { use((if ok { 1 } else { 2 }) + 1) }`,
		`func f(ok bool) { var x = if ok { if !ok { 1 } else { 2 } } else { 3 }; use(x) }`,
		`func f(ok bool) { var x = if ok { 1 } else if !ok { 2 } else { 3 }; use(x) }`,
		`func f(ok bool) { var x = if func() bool { return ok }() { 1 } else { 2 }; use(x) }`,
		`func f(ok bool) error { use(if ok { next()! } else { next()!! }); return nil }`,
		`func f(ok bool) { L: if ok { goto L }; use(if ok { 1 } else { 2 }) }`,
	} {
		t.Run(source, func(t *testing.T) {
			input := "package p\n" + source
			set := token.NewFileSet()
			parsed, err := ParseFile(set, "main.tgo", "main.tgo", []byte(input), make(map[string]bool))
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.Conditionals) == 0 {
				t.Fatal("no conditional expressions")
			}
			for _, pos := range parsed.Conditionals {
				if input[pos.Offset:pos.Offset+2] != "if" {
					t.Fatalf("conditional position: %v", pos)
				}
			}
			ast.Inspect(parsed.File, func(node ast.Node) bool {
				id, ok := node.(*ast.Ident)
				if ok && id.Name == "use" && !strings.HasPrefix(id.Name, "tgo") {
					pos := set.Position(id.Pos())
					if input[pos.Offset:pos.Offset+len(id.Name)] != id.Name {
						t.Errorf("source position: %v", pos)
					}
				}
				return true
			})
		})
	}
}

func TestPropagationSourcePositionsAndComments(t *testing.T) {
	input := "package p\nfunc f() error {\n x := next()! // keep this comment\n _ = x\n return nil\n}\n"
	set := token.NewFileSet()
	parsed, err := ParseFile(set, "main.tgo", "main.tgo", []byte(input), make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	ast.Inspect(parsed.File, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		if _, found := parsed.Propagations[id.Name]; found {
			count++
			pos, end := set.Position(call.Pos()), set.Position(call.End())
			if input[pos.Offset:end.Offset] != "next()!" {
				t.Fatalf("propagation source span: %v to %v", pos, end)
			}
		}
		return true
	})
	if count != 1 {
		t.Fatalf("propagation calls: %d", count)
	}
	if len(parsed.File.Comments) != 1 || parsed.File.Comments[0].List[0].Text != "// keep this comment" {
		t.Fatalf("source comments: %v", parsed.File.Comments)
	}
}
