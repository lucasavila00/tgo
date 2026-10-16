package frontend

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/scanner"
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
		"package p\nfunc f() error { _ = call() /* operand */ !!; return nil }",
		"package p\nfunc f() error { _ = (outer(inner() /* operand */ !) ) /* outer */ !!; return nil }",
		"package p\nfunc f(ok bool) error { _ = if ok { call()! } else { call()!! }; return nil }",
		"package p\nfunc f(n int) error { _ = switch n { case 1: call()!; default: call()!! }; return nil }",
		"package p\nfunc f(xs []int) error { _ = []int{call(x)!! for _, x := range xs}; return nil }",
	} {
		t.Run(input, func(t *testing.T) {
			checkPropagationPositions(t, input)
		})
	}
}

// TestParseErrorKeepsOriginalPosition checks a parser error position after a postfix operator.
func TestParseErrorKeepsOriginalPosition(t *testing.T) {
	input := "package p\nfunc f() { _ = call()! + }\n"
	_, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", []byte(input))
	if err == nil || !strings.Contains(err.Error(), "main.tgo:2:26:") {
		t.Fatalf("diagnostic: %v", err)
	}
}

// checkPropagationPositions checks each propagation node and its source call name.
func checkPropagationPositions(t *testing.T, input string) {
	t.Helper()
	fset := token.NewFileSet()
	checkPropagationPositionsInFileSet(t, fset, input)
}

// checkPropagationPositionsInFileSet checks propagation spans with the existing file-set base.
func checkPropagationPositionsInFileSet(t *testing.T, fset *token.FileSet, input string) {
	t.Helper()
	parsed, err := ParseFile(fset, "main_tgo.go", "main.tgo", []byte(input))
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	ast.Inspect(parsed.File, func(node ast.Node) bool {
		found += checkPropagationCallPosition(t, input, fset, parsed, node)
		return true
	})
	if found != len(parsed.CallNames) {
		t.Fatalf("nodes: %d; call names: %d", found, len(parsed.CallNames))
	}
}

// checkPropagationCallPosition checks one propagation span and its physical and adjusted source positions.
// It returns one for a checked propagation and zero for other nodes.
func checkPropagationCallPosition(t *testing.T, input string, fset *token.FileSet, parsed *Parsed, node ast.Node) int {
	t.Helper()
	operation, ok := node.(*ast.PropagateExpr)
	if !ok {
		return 0
	}
	file := fset.File(operation.Pos())
	if file == nil {
		t.Fatalf("propagation has no source file: %#v", operation)
	}
	if _, found := parsed.CallNames[operation]; !found {
		t.Fatal("propagation has no source call name")
	}
	position := fset.Position(operation.OpPos)

	width := 1
	if operation.Wrap {
		width = 2
	}
	if input[position.Offset:position.Offset+width] != strings.Repeat("!", width) {
		t.Fatalf("operator offset: %v", position)
	}
	if file.Offset(operation.End()) != position.Offset+width {
		t.Fatalf("node end: %d; operator end: %d", file.Offset(operation.End()), position.Offset+width)
	}
	physical := fset.PositionFor(operation.End()-token.Pos(width), false)
	adjusted := fset.Position(operation.End() - token.Pos(width))
	if physical.Filename != "main_tgo.go" || adjusted != position {
		t.Fatalf("operator position: %v; want %v; physical: %v", adjusted, position, physical)
	}
	if operation.X.Pos() < operation.Pos() || operation.X.End() > operation.End() {
		t.Fatal("operand is outside propagation span")
	}
	return 1
}

// TestParsePropagationInMultipleFiles checks that original byte offsets do not become global token positions.
func TestParsePropagationInMultipleFiles(t *testing.T) {
	fset := token.NewFileSet()
	for _, input := range []string{
		"package p\nfunc first() error { call()!; return nil }",
		"package p\nfunc second() error { outer(inner()!) /* operand */ !!; return nil }",
	} {
		checkPropagationPositionsInFileSet(t, fset, input)
	}
}

// TestParseMalformedPostfix checks that separated or extra ! bytes remain invalid syntax.
func TestParseMalformedPostfix(t *testing.T) {
	for _, suffix := range []string{"!!!", "! !", "!/* split */!"} {
		t.Run(suffix, func(t *testing.T) {
			input := "package p\nfunc f() { call()" + suffix + "; return }"
			_, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", []byte(input))
			if err == nil || !strings.Contains(err.Error(), "main.tgo:2:") {
				t.Fatalf("diagnostic: %v", err)
			}
			diagnostic := err.(scanner.ErrorList)[0]
			if diagnostic.Pos.Offset != strings.LastIndex(input, "!") {
				t.Fatalf("diagnostic position: %v", diagnostic.Pos)
			}
		})
	}
}

// TestParseMalformedGo checks that incomplete ordinary Go returns a diagnostic.
func TestParseMalformedGo(t *testing.T) {
	for _, input := range []string{"", "package", "func f() {}", "package p\nvar x = "} {
		t.Run(input, func(t *testing.T) {
			_, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", []byte(input))
			if err == nil {
				t.Fatalf("source diagnostic: %v", err)
			}
		})
	}
}
