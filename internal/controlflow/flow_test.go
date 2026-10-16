package controlflow_test

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/frontend/parser"
	"go/token"
	"testing"

	"github.com/lucasavila00/tgo/internal/controlflow"
)

// TestBranchPaths checks final-statement and branch-end reachability across jumps, loops, switches, and calls.
func TestBranchPaths(t *testing.T) {
	for _, test := range []struct {
		name, source string
		tail, end    bool
	}{
		{"value", `value()`, true, true},
		{"return", `return; value()`, false, false},
		{"partial_return", `if flag {return}; value()`, true, true},
		{"both_return", `if flag {return} else {return}; value()`, false, false},
		{"goto_tail", `goto tail; return; tail: value()`, true, true},
		{"goto_cycle", `again: goto again; value()`, false, false},
		{"inner_break", `for {break}; value()`, true, true},
		{"inner_continue", `for {continue}; value()`, false, false},
		{"label_break", `loop: for {break loop}; value()`, true, true},
		{"label_continue", `loop: for {continue loop}; value()`, false, false},
		{"loop_condition", `for flag {continue}; value()`, true, true},
		{"range_may_be_empty", `for range values {return}; value()`, true, true},
		{"switch_no_default", `switch n {case 1: return}; value()`, true, true},
		{"switch_returns", `switch n {case 1: return; default: return}; value()`, false, false},
		{"switch_break", `switch n {case 1: break; default: return}; value()`, true, true},
		{"switch_fallthrough", `switch n {case 1: fallthrough; default: return}; value()`, false, false},
		{"type_switch_returns", `switch n.(type) {case int: return; default: return}; value()`, false, false},
		{"empty_select", `select {}; value()`, false, false},
		{"select_returns", `select {case <-ch: return; default: return}; value()`, false, false},
		{"parenthesized_select_receive", `select {case (<-ch): return}; value()`, false, false},
		{"parenthesized_select_assignment", `select {case v = ((<-ch)): return}; value()`, false, false},
		{"select_break", `select {case <-ch: break; default: return}; value()`, true, true},
		{"outer_break", `break; value()`, false, false},
		{"outer_continue", `continue; value()`, false, false},
		{"outer_goto", `goto outside; value()`, false, false},
		{"closure_return", `func() {return}(); value()`, true, true},
		{"defer_panic", `defer panic("stop"); value()`, true, true},
		{"go_panic", `go panic("stop"); value()`, true, true},
		{"panic", `panic("stop"); value()`, false, false},
		{"short_circuit", `flag && panic("stop"); value()`, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "source.go", "package p; func f(){"+test.source+"}", 0)
			if err != nil {
				t.Fatal(err)
			}
			body := file.Decls[0].(*ast.FuncDecl).Body
			result := controlflow.Branch(body, controlflow.Syntax{Terminates: func(call *ast.CallExpr) bool {
				id, ok := call.Fun.(*ast.Ident)
				return ok && id.Name == "panic"
			}})
			if result.Tail != test.tail || result.End != test.end {
				t.Fatalf("got %+v; want tail=%t end=%t", result, test.tail, test.end)
			}
		})
	}
}

// TestNativeBranchPaths checks native values and generator sources in execution order.
func TestNativeBranchPaths(t *testing.T) {
	for _, test := range []struct {
		name, source string
		tail, end    bool
	}{
		{"if_returns", `_ = if flag {return} else {return}; value()`, false, false},
		{"if_partial_return", `_ = if flag {return} else {1}; value()`, true, true},
		{"else_if_returns", `_ = if flag {return} else if flag {return} else {return}; value()`, false, false},
		{"switch_returns", `_ = switch flag {case true:return;default:return}; value()`, false, false},
		{"short_return", `return ..., err; value()`, false, false},
		{"comprehension_empty", `_ = []int{if flag {return} else {return} for x:=range values}; value()`, true, true},
		{"comprehension_source_returns", `_ = []int{x for x:=range if flag {return} else {return}}; value()`, false, false},
		{"comprehension_later_source_returns",
			`_ = []int{x+y for x:=range values for y:=range if flag {return} else {return}}; value()`, true, true},
		{"comprehension_source_continue",
			`_ = []int{x for x:=range if flag {continue} else {continue}}; value()`, false, false},
		{"comprehension_next_source_continue",
			`_ = []int{x+y for x:=range values for y:=range if flag {continue} else {continue}}; value()`, true, true},
		{"native_closure_return", `_ = if flag {func() {return}(); 1} else {2}; value()`, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			file, err := parser.ParseTGoFile(token.NewFileSet(), "source.tgo", "source.tgo",
				[]byte("package p; func f(){"+test.source+"}"), 0)
			if err != nil {
				t.Fatal(err)
			}
			result := controlflow.Branch(file.Decls[0].(*ast.FuncDecl).Body, controlflow.Syntax{})
			if result.Tail != test.tail || result.End != test.end {
				t.Fatalf("got %+v; want tail=%t end=%t", result, test.tail, test.end)
			}
		})
	}
}

// TestExhaustiveSwitchPaths treats coverage labels as checks, not executable branches.
func TestExhaustiveSwitchPaths(t *testing.T) {
	source := `package p; func f() { switch account.(type) {
 case Account.Personal: return
 case Account.Business: return
 exhaustive:
 }; value() }`
	file, err := parser.ParseTGoFile(token.NewFileSet(), "source.tgo", "source.tgo", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	body := file.Decls[0].(*ast.FuncDecl).Body
	result := controlflow.Branch(body, controlflow.Syntax{})
	if result.Tail || result.End {
		t.Fatalf("exhaustive return reached continuation: %+v", result)
	}
}
