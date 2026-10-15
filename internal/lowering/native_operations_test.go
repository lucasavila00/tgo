package lowering

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker"
	"golang.org/x/tools/go/packages"
)

// TestNativeComprehensionScopes checks names after generator bindings move to fresh Go locals.
func TestNativeComprehensionScopes(t *testing.T) {
	input := `package main
import (
 "errors"
 "fmt"
)
type Number = int
var failure = errors.New("broken")
func read(index int) (int, error) {
 if index == 1 { return 0, failure }
 return 10, nil
}
func propagated() ([]int, error) {
 values := []int{10,20}
 return []int{fmt + read(nil)!! for fmt := range 1 for nil := range values}, nil
}
func generic[T ~int](values []T) []T {
 return []T{value for T, value := range values if T >= 0}
}
func main() {
 values := []int{10,20}
 fmt.Println([]int{int + value for int, value := range values})
 fmt.Println(generic(values))
 callbacks := []func()Number{func()Number{return int + value} for int, value := range values}
 fmt.Println(callbacks[0](), callbacks[1]())
 result, err := propagated()
 fmt.Println(result, err, errors.Is(err, failure))
}
`
	checkNativeOutput(t, input, "[10 21]\n[10 20]\n10 21\n[] read: broken true\n")
}

// TestNativeNestedPropagation checks each error and both ways to use the success results.
func TestNativeNestedPropagation(t *testing.T) {
	input := `package main
import (
 "errors"
 "fmt"
)
var first = errors.New("first")
var last = errors.New("last")
func read(which int) (int, error, error) {
 fmt.Print("R")
 if which == 1 { return 0, first, nil }
 if which == 2 { return 0, nil, last }
 return 7, nil, nil
}
func value(which int) (int,error) { return (read(which)!)!, nil }
func discarded(which int) error { (read(which)!)!; fmt.Print("D"); return nil }
func wrapped(which int) error { _ = (read(which)!!)!!; return nil }
func main() {
 for which := range 3 {
  result, err := value(which)
  fmt.Println(result, err == first, err == last)
  err = discarded(which)
  fmt.Println(err == first, err == last)
  err = wrapped(which)
  fmt.Println(err, errors.Is(err, first), errors.Is(err, last))
 }
}
`
	checkNativeOutput(t, input, "R7 false false\nRDfalse false\nR<nil> false false\n"+
		"R0 true false\nRtrue false\nRread: first true false\n"+
		"R0 false true\nRfalse true\nRread: last false true\n")
}

// TestNativeTupleContexts checks saved result types through assignments, calls, and returns.
func TestNativeTupleContexts(t *testing.T) {
	input := `package main
import "fmt"
func pair[T ~int](value T) (T,T,error) { fmt.Print("P"); return value,value+1,nil }
func consume(left,right int) int { return left+right }
func sum(values ...int) int { result:=0; for _,value:=range values { result+=value }; return result }
func returned(value int) (int,int,error) { return pair(value)!, ... }
func nested[T ~int](value T) (T,error) {
 return (func()(T,error,error){ fmt.Print("N"); return value,nil,nil }()!)!, ...
}
func run() error {
 left,right := pair(2)!
 fmt.Println(left,right)
 left,right = pair(3)!
 fmt.Println(left,right)
 fmt.Println(consume(pair(4)!))
 fmt.Println(sum(pair(5)!))
 fmt.Println(returned(6))
 fmt.Println(nested(8))
 return nil
}
func main() { fmt.Println(run()) }
`
	checkNativeOutput(t, input, "P2 3\nP3 4\nP9\nP11\nP6 7 <nil>\nN8 <nil>\n<nil>\n")
}

// TestNativeRangeOperands checks evaluated array expressions and uncalled closure bodies.
func TestNativeRangeOperands(t *testing.T) {
	input := `package main
import "fmt"
func main() {
 effects := 0
 for range if true { effects++; [2]int{} } else { [2]int{} } {}
 fmt.Println(effects)
 values := []int{i for i := range if true { effects++; [2]int{} } else { [2]int{} }}
 fmt.Println(values, effects)
 for range switch 1 { case 1: effects++; [2]int{}; default: [2]int{} } {}
 fmt.Println(effects)
 values = []int{i for i := range switch 1 { case 1: effects++; [2]int{}; default: [2]int{} }}
 fmt.Println(values, effects)
 for range [1]int{if true { effects++; 1 } else { 0 }} {}
 values = []int{i for i := range [1]int{switch 1 {case 1: effects++; 1; default: 0}}}
 fmt.Println(values, effects)
 for range [2]func()(int,error){func()(int,error){
  effects++
  values := []int{x for x := range 1}
  value := if true { effects++; values[0] } else { 0 }
  value = switch value { case 0: effects++; 1; default: 2 }
  if value != 0 { return ..., fmt.Errorf("unused") }
  return func()(int,error){ effects++; return value,nil }()!, ...
 }, nil} {}
 values = []int{i for i := range [2]func()(int,error){func()(int,error){
  effects++
  values := []int{x for x := range 1}
  value := if true { effects++; values[0] } else { 0 }
  value = switch value { case 0: effects++; 1; default: 2 }
  if value != 0 { return ..., fmt.Errorf("unused") }
  _ = func()(int,error){ effects++; return value,nil }()!
  return value, ...
 }, nil}}
 fmt.Println(values, effects)
}
`
	checkNativeOutput(t, input, "1\n[0 1] 2\n3\n[0 1] 4\n[0] 6\n[0 1] 6\n")
}

// TestNativeGotoScope checks that a goto can skip value-expression locals.
func TestNativeGotoScope(t *testing.T) {
	input := `package main
import "fmt"
func ifGoto() int {
 goto done
 _ = if x:=1; true {x} else {x}
 done: return 1
}
func switchGoto() int {
 goto done
 _ = switch x:=1; x {case 1: x; default: 0}
 done: return 2
}
func typeGoto() int {
 goto done
 _ = switch x:=any(1).(type) {case int: x; default: 0}
 done: return 3
}
func main(){ fmt.Println(ifGoto(),switchGoto(),typeGoto()) }
`
	checkNativeOutput(t, input, "1 2 3\n")
}

// TestNativeRangeSites checks which array operands can accept inserted statements.
func TestNativeRangeSites(t *testing.T) {
	input := `package main
func main() {
 effects:=0
 for range if true {effects++; [2]int{}} else {[2]int{}} {}
 for range switch 1 {case 1: effects++; [2]int{}; default: [2]int{}} {}
 _=[]int{i for i:=range if true {effects++; [2]int{}} else {[2]int{}}}
 for range [2]int{} {}
 for range [2]func(){func(){_=if true {effects++;1} else {0}},nil} {}
 _=[]int{i for i:=range [2]int{}}
 _=effects
}
`
	source, _ := loadNativeSource(t, input)
	dynamic, skipped := 0, 0
	ast.Inspect(source.inventory.Checked.Syntax[0], func(node ast.Node) bool {
		var operand ast.Expr
		switch node := node.(type) {
		case *ast.RangeStmt:
			operand = node.X
		case *ast.RangeClause:
			operand = node.X
		default:
			return true
		}
		expected := "range expression is not evaluated"
		switch operand.(type) {
		case *ast.IfExpr, *ast.SwitchExpr:
			expected = ""
			dynamic++
		default:
			skipped++
		}
		found := false
		for site, expression := range source.inventory.Expressions {
			if expression == operand {
				found = true
				if site.Reason != expected {
					t.Errorf("range operand %T: got %q, want %q", operand, site.Reason, expected)
				}
			}
		}
		if !found {
			t.Errorf("range operand %T has no recorded site", operand)
		}
		return true
	})
	if dynamic != 3 || skipped != 3 {
		t.Fatalf("checked %d dynamic and %d elided operands", dynamic, skipped)
	}
}

// loadNativeSource checks TGo with private syntax and types.
func loadNativeSource(t *testing.T, input string) (*Source, string) {
	t.Helper()
	directory := t.TempDir()
	filename := filepath.Join(directory, "main_tgo.go")
	for name, content := range map[string]string{
		"go.mod": "module example.com/native\n\ngo 1.27.1\n", "main_tgo.go": input,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	metadata := &packages.Package{ID: "example.com/native", PkgPath: "example.com/native", GoFiles: []string{filename}}
	roots, err := packages.Load(&packages.Config{Dir: directory, Mode: packages.NeedName |
		packages.NeedImports | packages.NeedDeps | packages.NeedExportFile}, ".")
	if err != nil {
		t.Fatal(err)
	}
	metadata.Imports = roots[0].Imports
	sizes, err := checker.TargetSizes(directory)
	if err != nil {
		t.Fatal(err)
	}
	loader := checker.NewLoader(directory, append(roots, metadata), map[string]string{"main_tgo.go": "main.tgo"}, sizes)
	checked, err := loader.Check(metadata)
	if err != nil {
		t.Fatal(err)
	}
	source, err := SourceFromCheckedPackage(checked, directory)
	if err != nil {
		t.Fatal(err)
	}
	return source, directory
}

// checkNativeOutput checks source immutability and runs emitted Go.
func checkNativeOutput(t *testing.T, input, expected string) {
	t.Helper()
	source, directory := loadNativeSource(t, input)
	checked := source.inventory.Checked
	filename := filepath.Join(directory, "main_tgo.go")
	var before bytes.Buffer
	if err := ast.Fprint(&before, checked.Fset, checked.Syntax[0], nil); err != nil {
		t.Fatal(err)
	}
	output, err := source.LowerPropagations()
	if err != nil {
		t.Fatal(err)
	}
	var after bytes.Buffer
	if err := ast.Fprint(&after, checked.Fset, checked.Syntax[0], nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Bytes(), after.Bytes()) {
		t.Fatal("native source changed during lowering")
	}
	if err := os.WriteFile(filename, output[filename], 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", ".")
	command.Dir = directory
	actual, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s\n%s", err, actual, output[filename])
	}
	if string(actual) != expected {
		t.Fatalf("got %q, want %q\n%s", actual, expected, output[filename])
	}
}

// TestNativeShortReturnCallees checks propagation, identity, and context for both return placeholders.
func TestNativeShortReturnCallees(t *testing.T) {
	input := `package main
import (
 "errors"
 "fmt"
)
var broken = errors.New("broken")
func first(err error) (int, error) {
 return func() (int, error) { return ..., err }()!, nil
}
func firstWrapped(err error) (int, error) {
 return func() (int, error) { return ..., err }()!!, nil
}
func last(err error) (int, error) {
 return func() (int, error) { if err != nil { return 0, err }; return 1, ... }()!, nil
}
func lastWrapped(err error) (int, error) {
 return func() (int, error) { if err != nil { return 0, err }; return 1, ... }()!!, nil
}
func main() {
 for _, err := range []error{nil, broken} {
  a, firstError := first(err)
  b, lastError := last(err)
  fmt.Println(a, b, firstError == err, lastError == err)
 }
 _, err := firstWrapped(broken)
 fmt.Println(errors.Is(err, broken), err.Error() == "func() (int, error) { return ..., err }: broken")
 _, err = lastWrapped(broken)
 fmt.Println(errors.Is(err, broken),
  err.Error() == "func() (int, error) { if err != nil { return 0, err }; return 1, ... }: broken")
 a, firstError := firstWrapped(nil)
 b, lastError := lastWrapped(nil)
 fmt.Println(a, b, firstError == nil, lastError == nil)
}
`
	checkNativeOutput(t, input, "0 1 true true\n0 0 true true\ntrue true\ntrue true\n0 1 true true\n")
}
