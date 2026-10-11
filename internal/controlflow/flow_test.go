package controlflow_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/lucasavila00/tgo/internal/controlflow"
)

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
