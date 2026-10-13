package frontend

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// TestParsePropagationPositions checks postfix positions with nesting, line directives, and non-ASCII input.
func TestParsePropagationPositions(t *testing.T) {
	for _, input := range []string{
		"package p\nfunc f() error { _ = call()!; return nil }",
		"package p\nfunc f() error { _ = outer(inner()!)!!; return nil }",
		"package p\n//line virtual.tgo:40\nfunc f() error { _ = outer(inner()!)!!; return nil }",
		"package p\nfunc f() error { _ = (call())!!; _ = call()!; return nil }",
		"package p\nfunc f() error { _ = call(\"é\")!; return nil }",
	} {
		t.Run(input, func(t *testing.T) {
			checkPropagationPositions(t, input)
		})
	}
}

// TestParseErrorKeepsOriginalPosition checks that a parser error refers to the input after postfix replacement.
func TestParseErrorKeepsOriginalPosition(t *testing.T) {
	input := "package p\nfunc f() { _ = call()! + }\n"
	_, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", []byte(input), make(map[string]bool))
	if err == nil || !strings.Contains(err.Error(), "main.tgo:2:26:") {
		t.Fatalf("diagnostic: %v", err)
	}
}

// checkPropagationPositions checks each parsed marker and requires one marker per propagation record.
func checkPropagationPositions(t *testing.T, input string) {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := ParseFile(fset, "main_tgo.go", "main.tgo", []byte(input), make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	ast.Inspect(parsed.File, func(node ast.Node) bool {
		found += checkPropagationCallPosition(t, input, fset, parsed, node)
		return true
	})
	if found != len(parsed.Propagations) {
		t.Fatalf("markers: %d; operations: %d", found, len(parsed.Propagations))
	}
}

// checkPropagationCallPosition checks one marker span and its physical and adjusted source positions.
// It returns one for a checked marker and zero for other nodes.
func checkPropagationCallPosition(t *testing.T, input string, fset *token.FileSet, parsed *Parsed, node ast.Node) int {
	t.Helper()
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return 0
	}
	file := fset.File(call.Pos())
	if file == nil {
		t.Fatalf("call has no source file: %#v", call)
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return 0
	}
	op, ok := parsed.Propagations[id.Name]
	if !ok {
		return 0
	}

	width := 1
	if op.Wrap {
		width = 2
	}
	if input[op.Position.Offset:op.Position.Offset+width] != strings.Repeat("!", width) {
		t.Fatalf("operator offset: %v", op.Position)
	}
	if file.Offset(call.End()) != op.Position.Offset+width {
		t.Fatalf("marker end: %d; operator end: %d", file.Offset(call.End()), op.Position.Offset+width)
	}
	physical := fset.PositionFor(call.End()-token.Pos(width), false)
	adjusted := fset.Position(call.End() - token.Pos(width))
	if physical.Filename != "main_tgo.go" || adjusted != op.Position {
		t.Fatalf("operator position: %v; want %v; physical: %v", adjusted, op.Position, physical)
	}
	if call.Args[0].Pos() < call.Pos() || call.Args[0].End() > call.End() {
		t.Fatal("operand is outside marker span")
	}
	return 1
}

// TestParseMalformedGo checks that incomplete ordinary Go returns a diagnostic.
func TestParseMalformedGo(t *testing.T) {
	for _, input := range []string{"", "package", "func f() {}", "package p\nvar x = "} {
		t.Run(input, func(t *testing.T) {
			_, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", []byte(input), make(map[string]bool))
			if err == nil {
				t.Fatalf("source diagnostic: %v", err)
			}
		})
	}
}
