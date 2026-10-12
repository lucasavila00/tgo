package frontend

import (
	"fmt"
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
		`func f(ok bool, xs []int) { use(xs[0:if ok {1} else {2}:if ok {2} else {3}]) }`,
		`func f(ok bool) { use((if ok { 1 } else { 2 }) + 1) }`,
		`func f(ok bool) { use(if ok { log(); (if !ok {1} else {"two"}) } else {log(); "three"}, any(4)) }`,
		`func f(ok bool) { switch { case ok: if ok { use(1) }; default: if ok { ` +
			`use(2) } }; use(if ok {1} else {2}) }`,

		`func f(ok bool) { var x = if ok { if !ok { 1 } else { 2 } } else { 3 }; use(x) }`,
		`func f(ok bool) { var x = if ok { 1 } else if !ok { 2 } else { 3 }; use(x) }`,
		`func f(ok bool) { var x = if func() bool { return ok }() { 1 } else { 2 }; use(x) }`,
		`func f(ok bool) error { use(if ok { next()! } else { next()!! }); return nil }`,
		`func f(ok bool) { L: if ok { goto L }; use(if ok { 1 } else { 2 }) }`,
		`func f() { turns:=0; again: for i:=if turns<2 {turns++;0} else {0}; i<1;` +
			`i++ { _=int(if turns<2 {goto again} else {break again}) } }`,
	} {
		t.Run(source, func(t *testing.T) {
			checkConditionalPositions(t, source)
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

func TestPropagationRequiresCall(t *testing.T) {
	for _, operator := range []string{"!", "!!"} {
		t.Run(operator, func(t *testing.T) {
			defer func() {
				err := recover()
				if err == nil || !strings.Contains(fmt.Sprint(err), "propagation requires a call") {
					t.Fatalf("diagnostic: %v", err)
				}
			}()
			input := "package p\nfunc f(err error) error { value := (err)" + operator + "; return value }"
			if _, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(input),
				make(map[string]bool)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConditionalNamesKeepSourceBindings(t *testing.T) {
	input := "package p\nfunc tgoConditional1() int {return 4}; func f() int { return if " +
		"true { tgoConditional1() } else { 0 } }"
	parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(input), make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	if _, collision := parsed.Conditionals["tgoConditional1"]; collision {
		t.Fatal("conditional marker replaced a source binding")
	}
}

func TestConditionalInsideStatementCondition(t *testing.T) {
	input := `package p
func checkCopy(ok bool) {
	type integers []int
	dst := integers{1, 2}
	calls := 0
	n := copy(dst, if ok { calls++; nil } else { calls++; nil })
	values := append(dst, (if ok { calls++; nil } else { calls++; nil })...)
	bytes := []byte{0, 0, 0}
	copied := copy(bytes, if ok { "one" } else { "two" })
	unchanged := append(bytes, (if ok { nil } else { nil })...)
	if copied != 3 || len(unchanged) != 3 || copy(bytes, if ok { nil } else { nil }) != 0 {
		panic("wrong byte operation")
	}
	if calls != 2 || n != 0 || len(values) != 2 || values[0] != 1 || values[1] != 2 {
		panic("wrong slice operation")
	}
}

`
	parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(input), make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Conditionals) != 5 {
		t.Fatalf("if values: %d, want 5", len(parsed.Conditionals))
	}
}

func TestConditionalBatchGrammar(t *testing.T) {
	input := `package p
func f(ok bool, xs []int) {
again:
 if ok { use(1); if !ok { goto again } }
 use(if ready := func() bool { if ok { return true }; return false }; ready() {
  (if ok { 1 } else { 2 })
 } else if func() bool { return !ok }() {
  3
 } else { 4 })
 use(xs[0:if ok { 1 } else { 2 }])
 mapping := map[string]int{"value": if ok { 5 } else { 6 }}
 _ = mapping
}
`
	parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(input), make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Conditionals) != 4 {
		t.Fatalf("if values: %d, want 4", len(parsed.Conditionals))
	}
	for _, position := range parsed.Conditionals {
		if input[position.Offset:position.Offset+2] != "if" {
			t.Fatalf("source position: %v", position)
		}
	}
}

func TestConditionalBatchIndependentOperands(t *testing.T) {
	var source strings.Builder
	source.WriteString("package p\nfunc f(ok bool) {\n")
	for i := 0; i < 64; i++ {
		source.WriteString("if ok { use(0) }; use(if ok { 1 } else { 2 });\n")
	}
	source.WriteString("}\n")
	parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(source.String()),
		make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Conditionals) != 64 {
		t.Fatalf("if values: %d, want 64", len(parsed.Conditionals))
	}
}

func TestParseSwitchExpressions(t *testing.T) {
	for _, source := range []string{
		`func f(n int) int {return switch n {case 1: 2; default: 3}}`,
		`func f(n int) {use(switch x:=n; x {case 1: x; default: 3})}`,
		`func f(n int) {use(switch {case n>0: 1; default: 0})}`,
		`func f(v any) {use(switch x:=v.(type) {case int: x; default: 0})}`,
		`func f(n int) {use(switch n {case 1: switch n {case 1: 2; default: 3}; default: 4})}`,
		`func f(n int) {use(if n>0 {switch n {case 1: 2; default: 3}} else {4})}`,
		`func f(n int) {use(switch n {case if n>0 {1} else {2}: 3; default: 4})}`,
		`func f(n int) {switch n {case 1: use(1); default: use(2)}; use(switch n {case 1: 2; default: 3})}`,
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
				if !strings.HasPrefix(input[pos.Offset:], "switch") && !strings.HasPrefix(input[pos.Offset:], "if") {
					t.Fatalf("source position: %v", pos)
				}
			}
		})
	}
}

func checkConditionalPositions(t *testing.T, source string) {
	t.Helper()
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
}
