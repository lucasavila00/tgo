package frontend

import (
	"github.com/lucasavila00/tgo/internal/ast"
	stockScanner "go/scanner"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

// TestPostfixLogicalSourcePositions checks logical diagnostics with physical AST file identity.
func TestPostfixLogicalSourcePositions(t *testing.T) {
	set := token.NewFileSet()
	set.AddFile("earlier.go", -1, 80)
	for _, sourceName := range []string{"main.tgo", "pkg/main.tgo"} {
		for _, directive := range []string{
			"", "//line virtual.tgo:50\n", "//line virtual.tgo:50:10\n",
			"//line /source/virtual.tgo:50:10\n", "//line :50:10\n", "//line :50\n",
			"//line virtual.tgo:50:10\n//line :70:5\n", "/*line virtual.tgo:50:10*/",
		} {
			for _, suffix := range []string{"!", "!!"} {
				t.Run(sourceName+directive+suffix, func(t *testing.T) {
					checkPostfixSourcePosition(t, set, sourceName, directive, suffix)
				})
			}
		}
	}
}

// checkPostfixSourcePosition compares one propagation node with stock logical line resolution.
func checkPostfixSourcePosition(t *testing.T, set *token.FileSet, sourceName, directive, suffix string) {
	t.Helper()
	input := "package p\n" + directive + "func run() error { _ = target(\"é\")" + suffix + "; return nil }"
	const physical = "/build/pkg/main_tgo.go"
	parsed, err := ParseFile(set, physical, sourceName, []byte(input))
	if err != nil {
		t.Fatal(err)
	}
	offset := strings.Index(input, suffix)
	want := scannedSourcePosition(input, sourceName, offset)
	for operation := range parsed.CallNames {
		if set.Position(operation.OpPos) != want {
			t.Fatalf("operator position: %v; want %v", set.Position(operation.OpPos), want)
		}
	}
	checkPhysicalAndLogicalPosition(t, set, parsed, physical, want, len(suffix))
}

// scannedSourcePosition uses Go's scanner with the logical file name as the diagnostic reference.
func scannedSourcePosition(input, sourceName string, offset int) token.Position {
	set := token.NewFileSet()
	file := set.AddFile(sourceName, -1, len(input))
	var scan stockScanner.Scanner
	scan.Init(file, []byte(input), nil, stockScanner.ScanComments)
	for {
		if _, kind, _ := scan.Scan(); kind == token.EOF {
			return file.Position(file.Pos(offset))
		}
	}
}

// checkPhysicalAndLogicalPosition checks that the source node uses the physical file and logical line information.
func checkPhysicalAndLogicalPosition(t *testing.T, set *token.FileSet, parsed *Parsed, physical string,
	want token.Position, width int) {
	t.Helper()
	found := false
	ast.Inspect(parsed.File, func(node ast.Node) bool {
		operation, ok := node.(*ast.PropagateExpr)
		if !ok {
			return true
		}
		found = true
		position := operation.End() - token.Pos(width)
		if set.File(position).Name() != physical || set.PositionFor(position, false).Filename != physical ||
			set.Position(position) != want {
			t.Fatalf("physical: %s; logical: %v; want: %v", set.File(position).Name(), set.Position(position), want)
		}
		return true
	})
	if !found {
		t.Fatal("missing propagation node")
	}
}

// TestTGoRecordsLogicalSourcePositions checks the same logical position map for each TGo record.
func TestTGoRecordsLogicalSourcePositions(t *testing.T) {
	for _, test := range []struct {
		body   string
		starts []string
	}{
		{"_ = if ok { target()! } else { 0 }; return nil", []string{"if", "!"}},
		{"_ = switch n { case 1: target()!!; default: 0 }; return nil", []string{"switch", "!!"}},
		{"_ = []int{target(x)!! for _, x := range xs}; return nil", []string{"[]int", "!!"}},
		{"return ..., target()!", []string{"...", "!"}},
	} {
		t.Run(test.body, func(t *testing.T) {
			const sourceName = "pkg/main.tgo"
			input := "package p\n//line virtual.tgo:70:2\nfunc run() error { " + test.body + " }"
			set := token.NewFileSet()
			parsed, err := ParseFile(set, "/build/pkg/main_tgo.go", sourceName,
				[]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			want := make(map[int]token.Position)
			for _, start := range test.starts {
				offset := strings.Index(input, start)
				want[offset] = scannedSourcePosition(input, sourceName, offset)
			}
			if positions := featureSourcePositions(set, parsed); !reflect.DeepEqual(positions, want) {
				t.Fatalf("positions: %v; want %v", positions, want)
			}
		})
	}
}

// TestParserLogicalDiagnosticPosition checks syntax errors against the same logical source map.
func TestParserLogicalDiagnosticPosition(t *testing.T) {
	const sourceName = "pkg/main.tgo"
	input := "package p\n//line virtual.tgo:70:2\nfunc run() error { _ = target()! + }"
	set := token.NewFileSet()
	parsed, err := ParseFile(set, "/build/pkg/main_tgo.go", sourceName, []byte(input))
	if parsed != nil || err == nil {
		t.Fatalf("parsed: %v; error: %v", parsed, err)
	}
	position := err.(stockScanner.ErrorList)[0].Pos
	want := scannedSourcePosition(input, sourceName, strings.LastIndex(input, "}"))
	if position != want {
		t.Fatalf("diagnostic position: %v; want %v", position, want)
	}
}

// featureSourcePositions collects the original token position of each native TGo form.
func featureSourcePositions(set *token.FileSet, parsed *Parsed) map[int]token.Position {
	positions := make(map[int]token.Position)
	ast.Inspect(parsed.File, func(node ast.Node) bool {
		var position token.Pos
		switch node := node.(type) {
		case *ast.PropagateExpr:
			position = node.OpPos
		case *ast.IfExpr:
			position = node.If
		case *ast.SwitchExpr:
			position = node.Pos()
		case *ast.ComprehensionExpr:
			position = node.Pos()
		case *ast.ShortReturnStmt:
			position = node.Placeholder
		}
		if position.IsValid() {
			mapped := set.Position(position)
			positions[mapped.Offset] = mapped
		}
		return true
	})
	return positions
}
