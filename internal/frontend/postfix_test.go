package frontend

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// TestPostfixOperands checks source call names before nested helper calls are constructed.
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
			parsed, err := ParseFile(set, "main_tgo.go", "main.tgo", []byte(input), make(map[string]bool))
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.Propagations) != len(test.names) {
				t.Fatalf("operators: %d, want %d", len(parsed.Propagations), len(test.names))
			}
			for index, name := range test.names {
				operation := parsed.Propagations[fmt.Sprintf("tgoPropagation%d", index+1)]
				if operation.CallName != name || operation.Wrap != test.wrap[index] {
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
			parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(input),
				make(map[string]bool))
			if parsed == nil || err != nil {
				t.Fatalf("parsed: %v; error: %v", parsed, err)
			}
		})
	}
}

// TestPostfixFailedParseNames checks that a failed attempt does not reserve helper names.
func TestPostfixFailedParseNames(t *testing.T) {
	used := make(map[string]bool)
	input := []byte("package p\nfunc run() error { target()!; _ = }")
	if _, err := ParseFile(token.NewFileSet(), "bad.tgo", "bad.tgo", input, used); err == nil {
		t.Fatal("invalid source parsed")
	}
	for name := range used {
		if strings.HasPrefix(name, "tgoPropagation") {
			t.Fatalf("failed parse reserved %s", name)
		}
	}
	input = []byte("package p\nfunc run() error { target()!; return nil }")
	parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", input, used)
	if err != nil || parsed.Propagations["tgoPropagation1"].Name != "tgoPropagation1" {
		t.Fatalf("parsed: %v; error: %v", parsed, err)
	}
}

// TestPostfixMixedSyntaxNames checks helper names and spans after preparation and parser recovery.
func TestPostfixMixedSyntaxNames(t *testing.T) {
	for _, expression := range []string{
		"if ok { target()! } else { target()!! }",
		"switch n { case 1: target()!; default: target()!! }",
		"[]int{target(x)!! for _, x := range xs if check(x)!}",
		"[]int{if ok { target()! } else { target()!! }}",
	} {
		t.Run(expression, func(t *testing.T) {
			input := "package p\nfunc run() error { _ = " + expression + "; return .. nil }"
			set := token.NewFileSet()
			used := make(map[string]bool)
			parsed, err := ParseFile(set, "main_tgo.go", "main.tgo", []byte(input), used)
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.Propagations) != 2 || !used["tgoPropagation1"] || !used["tgoPropagation2"] ||
				used["tgoPropagation3"] {
				t.Fatalf("operations: %+v; names: %v", parsed.Propagations, used)
			}
			checkPropagationPositions(t, input)
		})
	}
}

// TestPostfixCallBinding checks that a suffix on a type assertion cannot select an earlier call.
func TestPostfixCallBinding(t *testing.T) {
	input := "package p\nfunc run() error { _ = factory().(error)!; return nil }"
	parsed, err := ParseFile(token.NewFileSet(), "main.tgo", "main.tgo", []byte(input), make(map[string]bool))
	if parsed != nil || err == nil || !strings.Contains(err.Error(), "propagation requires a call") {
		t.Fatalf("parsed: %v; error: %v", parsed, err)
	}
}

// TestPostfixMultipleFiles checks source positions and helper names across file-set bases.
func TestPostfixMultipleFiles(t *testing.T) {
	set, used := token.NewFileSet(), make(map[string]bool)
	for _, filename := range []string{"first.tgo", "second.tgo"} {
		input := "package p\nfunc run() error { _ = (target()) /* source */ !!; return nil }"
		parsed, err := ParseFile(set, filename+".go", filename, []byte(input), used)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed.File, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			operation, exists := parsed.Propagations[name.Name]
			if exists && (set.Position(call.End()).Offset != strings.Index(input, "!!")+2 ||
				operation.Position.Filename != filename) {
				t.Fatalf("call end: %v; operator: %+v", set.Position(call.End()), operation)
			}
			return true
		})
	}
	if !used["tgoPropagation1"] || !used["tgoPropagation2"] {
		t.Fatalf("names: %v", used)
	}
}
