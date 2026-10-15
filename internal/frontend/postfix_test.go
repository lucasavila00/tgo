package frontend

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// TestPostfixOperands checks source call context on the actual nested propagation nodes.
func TestPostfixOperands(t *testing.T) {
	for _, test := range []struct {
		expression string
		names      []string
		wrap       []bool
	}{
		{"target()!", []string{"target"}, []bool{false}},
		{"target()!!", []string{"target"}, []bool{true}},
		{"target[int]()!", []string{"target[int]"}, []bool{false}},
		{"(target())!!", []string{"target"}, []bool{true}},
		{"value.Fetch()!", []string{"value.Fetch"}, []bool{false}},
		{"outer(inner()!)!!", []string{"inner", "outer"}, []bool{false, true}},
		{"factory()!()!!", []string{"factory", "factory()"}, []bool{false, true}},
		{"(factory()!)!", []string{"factory", "factory"}, []bool{false, false}},
	} {
		t.Run(test.expression, func(t *testing.T) {
			input := "package p\nfunc run() error { _ = " + test.expression + "; return nil }"
			set := token.NewFileSet()
			parsed, err := ParseFile(set, "main_tgo.go", "main.tgo", []byte(input))
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.CallNames) != len(test.names) {
				t.Fatalf("operators: %d, want %d", len(parsed.CallNames), len(test.names))
			}
			for index, name := range test.names {
				operation := orderedPropagations(parsed)[index]
				if parsed.CallNames[operation] != name || operation.Wrap != test.wrap[index] {
					t.Fatalf("operator %d: %+v", index, operation)
				}
			}
			checkPropagationPositions(t, input)
		})
	}
}

// TestPostfixNewlines checks statement endings and ordinary unary operators with newlines.
func TestPostfixNewlines(t *testing.T) {
	for _, body := range []string{
		"target()!\nreturn nil", "target()!!\nreturn nil", "target()! // keep\nreturn nil",
		"target()!! /* keep\ncomment */ return nil", "_ = !flag\nreturn nil",
		"_ = !\nflag\nreturn nil", "_ = target() != nil\nreturn nil",
		"_ = func() error { return nil }()!\nreturn nil",
	} {
		t.Run(body, func(t *testing.T) {
			input := "package p\nfunc run() error {\n" + body + "\n}"
			parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(input))
			if parsed == nil || err != nil {
				t.Fatalf("parsed: %v; error: %v", parsed, err)
			}
		})
	}
}

// TestPostfixFailedParseReturnsNoSyntax checks that a failed parse does not publish a partial source tree.
func TestPostfixFailedParseReturnsNoSyntax(t *testing.T) {
	input := []byte("package p\nfunc run() error { target()!; _ = }")
	if parsed, err := ParseFile(token.NewFileSet(), "bad.tgo", "bad.tgo", input); parsed != nil || err == nil {
		t.Fatalf("parsed: %v; diagnostic: %v", parsed, err)
	}
	input = []byte("package p\nfunc run() error { target()!; return nil }")
	parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", input)
	if err != nil || len(parsed.CallNames) != 1 {
		t.Fatalf("parsed: %v; error: %v", parsed, err)
	}
}

// TestPostfixMixedSyntaxNames checks native propagation identity and spans across mixed TGo syntax.
func TestPostfixMixedSyntaxNames(t *testing.T) {
	for _, expression := range []string{
		"if ok { target()! } else { target()!! }",
		"switch n { case 1: target()!; default: target()!! }",
		"[]int{target(x)!! for _, x := range xs if check(x)!}",
		"[]int{if ok { target()! } else { target()!! }}",
	} {
		t.Run(expression, func(t *testing.T) {
			input := "package p\nfunc run() error { _ = " + expression + "; return ..., nil }"
			set := token.NewFileSet()
			parsed, err := ParseFile(set, "main_tgo.go", "main.tgo", []byte(input))
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.CallNames) != 2 {
				t.Fatalf("source call names: %v", parsed.CallNames)
			}
			checkPropagationPositions(t, input)
		})
	}
}

// TestPostfixCallBinding checks that a suffix on a type assertion cannot select an earlier call.
func TestPostfixCallBinding(t *testing.T) {
	input := "package p\nfunc run() error { _ = factory().(error)!; return nil }"
	parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(input))
	if parsed != nil || err == nil || !strings.Contains(err.Error(), "propagation requires a call") {
		t.Fatalf("parsed: %v; error: %v", parsed, err)
	}
}

// TestPostfixMultipleFiles checks source positions and call context across file-set bases.
func TestPostfixMultipleFiles(t *testing.T) {
	set := token.NewFileSet()
	for _, filename := range []string{"first.tgo", "second.tgo"} {
		input := "package p\nfunc run() error { _ = (target()) /* source */ !!; return nil }"
		parsed, err := ParseFile(set, filename+".go", filename, []byte(input))
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed.File, func(node ast.Node) bool {
			operation, ok := node.(*ast.PropagateExpr)
			if !ok {
				return true
			}
			if set.Position(operation.End()).Offset != strings.Index(input, "!!")+2 ||
				set.Position(operation.OpPos).Filename != filename {
				t.Fatalf("node end: %v; operator: %v", set.Position(operation.End()), set.Position(operation.OpPos))
			}
			return true
		})
	}
}

// orderedPropagations returns native operators in source token order for call-context assertions.
func orderedPropagations(parsed *Parsed) []*ast.PropagateExpr {
	var operations []*ast.PropagateExpr
	for expression := range parsed.CallNames {
		operations = append(operations, expression)
	}
	sort.Slice(operations, func(first, second int) bool { return operations[first].OpPos < operations[second].OpPos })
	return operations
}
