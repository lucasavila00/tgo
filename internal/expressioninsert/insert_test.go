package expressioninsert

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBeforeReturnRunsAtSelectedOperand(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func value() string { fmt.Print("value "); return "value" }
func run() string { return "left " + value() }
func main() { fmt.Print(run()) }
`, "value", `return "before"`, true)
	if output != "before" {
		t.Fatalf("output = %q", output)
	}
}

func TestAfterReturnUsesCompletedValueAndSkipsLaterWork(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func left() string { fmt.Print("left "); return "L" }
func value() string { fmt.Print("value "); return "V" }
func right() string { fmt.Print("right "); return "R" }
func run() string { return left() + value() + right() }
func main() { fmt.Print(run()) }
`, "value", `return "after"`, false)
	if output != "left value after" {
		t.Fatalf("output = %q", output)
	}
}

func TestShortCircuitSkipsHookWithExpression(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func selected() bool { fmt.Print("selected "); return true }
func run(flag bool) string {
	if flag && selected() { return "body" }
	return "done"
}
func main() { fmt.Print(run(false), " ", run(true)) }
`, "selected", `return "hook"`, true)
	if output != "done hook" {
		t.Fatalf("output = %q", output)
	}
}

func TestAfterDoesNotRunOnPanic(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func selected() int { fmt.Print("selected "); panic("boom") }
func run() { defer func() { fmt.Print(recover()) }(); _ = selected() }
func main() { run() }
`, "selected", `fmt.Print("after ")`, false)
	if output != "selected boom" {
		t.Fatalf("output = %q", output)
	}
}

func TestDirectRecoverKeepsItsFrame(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func run() {
	defer func() {
		value := recover()
		fmt.Print(value)
	}()
	panic("boom")
}
func main() { run() }
`, "recover", `if false { return }`, false)
	if output != "boom" {
		t.Fatalf("output = %q", output)
	}
}

func TestUntypedConstantKeepsAssignmentContext(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func run() string { var value float64 = 1; return fmt.Sprint(value) }
func main() { fmt.Print(run()) }
`, "1", `if false { return "hook" }`, true)
	if output != "1" {
		t.Fatalf("output = %q", output)
	}
}

func TestAfterHookCannotChangeCapturedResult(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func selected(value int) bool { return value == 0 }
func run() string {
	x := 0
	if selected(x) { return "hit" }
	return "miss"
}
func main() { fmt.Print(run()) }
`, "selected", `x = 1`, false)
	if output != "hit" {
		t.Fatalf("output = %q", output)
	}
}

func TestConstantLenOperandGetsNoRuntimeHook(t *testing.T) {
	file, info, target := parseTyped(t, `package sample
func run() int {
	x := 2
	return len([1]int{x})
}
`, "ident:x:last")
	inserted, err := Insert(file, info, target, []ast.Stmt{parseStatement(t, `return 2`)}, nil)
	if inserted || err != nil {
		t.Fatalf("Insert = %v, %v", inserted, err)
	}
}

func TestAssignmentDestinationRunsBeforeSelectedRightSide(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func key() int { fmt.Print("key "); return 0 }
func selected() int { fmt.Print("selected "); return 1 }
func run() { values := map[int]int{}; values[key()] = selected() }
func main() { run() }
`, "selected", `return`, false)
	if output != "key selected " {
		t.Fatalf("output = %q", output)
	}
}

func TestSelectedArrayLocationDoesNotMoveBoundsPanicBeforeRightSide(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func right() int { fmt.Print("right "); return 1 }
func index() int { return 2 }
func run() {
	defer func() { fmt.Print(recover() != nil) }()
	values := [1]int{}
	values[index()] = right()
}
func main() { run() }
`, "indexexpr", `fmt.Print("after ")`, false)
	if output != "after right true" {
		t.Fatalf("output = %q", output)
	}
}

func TestTupleResultIsCapturedBeforeAfterReturn(t *testing.T) {
	output := transformAndRun(t, `package main
import "fmt"
func pair() (int, int) { fmt.Print("pair "); return 1, 2 }
func consume(left, right int) string { fmt.Print("consume "); return fmt.Sprint(left + right) }
func run() string { return consume(pair()) }
func main() { fmt.Print(run()) }
`, "pair", `return "hook"`, false)
	if output != "pair hook" {
		t.Fatalf("output = %q", output)
	}
}

func TestRootDeferredCallHasNoPureGoLowering(t *testing.T) {
	file, info, target := parseTyped(t, `package sample
func catcher() {}
func run() { defer catcher() }
`, "catcher")
	inserted, err := Insert(file, info, target, nil, []ast.Stmt{parseStatement(t, `return`)})
	if inserted || err == nil || !strings.Contains(err.Error(), "root deferred call") {
		t.Fatalf("Insert = %v, %v", inserted, err)
	}
}

func transformAndRun(t *testing.T, source, selected, hook string, before bool) string {
	t.Helper()
	file, info, target := parseTyped(t, source, selected)
	statement := parseStatement(t, hook)
	var beforeStatements, afterStatements []ast.Stmt
	if before {
		beforeStatements = []ast.Stmt{statement}
	} else {
		afterStatements = []ast.Stmt{statement}
	}
	inserted, err := Insert(file, info, target, beforeStatements, afterStatements)
	if err != nil || !inserted {
		t.Fatalf("Insert = %v, %v", inserted, err)
	}
	var generated bytes.Buffer
	if err := format.Node(&generated, token.NewFileSet(), file); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "main.go")
	if err := os.WriteFile(path, generated.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", path)
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run: %v\n%s\n%s", err, data, generated.String())
	}
	return string(data)
}

func parseTyped(t *testing.T, source, selected string) (*ast.File, *types.Info, ast.Expr) {
	t.Helper()
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "main.go", source, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	configuration := types.Config{Importer: importer.Default()}
	if _, err := configuration.Check("sample", files, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	var target ast.Expr
	ast.Inspect(file, func(node ast.Node) bool {
		if target != nil && !strings.HasSuffix(selected, ":last") {
			return false
		}
		switch expression := node.(type) {
		case *ast.CallExpr:
			name, ok := expression.Fun.(*ast.Ident)
			if ok && name.Name == selected {
				target = expression
				return false
			}
		case *ast.BasicLit:
			if expression.Value == selected {
				target = expression
				return false
			}
		case *ast.Ident:
			if (selected == "ident:"+expression.Name || selected == "ident:"+expression.Name+":last") && info.Types[expression].IsValue() {
				target = expression
				return false
			}
		case *ast.IndexExpr:
			if selected == "indexexpr" {
				target = expression
				return false
			}
		}
		return true
	})
	if target == nil {
		t.Fatalf("did not find %q", selected)
	}
	return file, info, target
}

func parseStatement(t *testing.T, source string) ast.Stmt {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "hook.go", "package hook\nfunc hook() { "+source+" }", 0)
	if err != nil {
		t.Fatal(err)
	}
	return file.Decls[0].(*ast.FuncDecl).Body.List[0]
}
