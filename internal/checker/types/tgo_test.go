package types_test

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

func checkTGo(t *testing.T, source string) (*types.Info, error) {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := frontend.ParseFile(fset, "main.tgo", "main.tgo", []byte("package p\n"+source), make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{TGo: make(map[string]types.TGoOperation)}
	for name, position := range parsed.Conditionals {
		config.TGo[name] = types.TGoOperation{Conditional: true, Position: position}
	}
	for name, op := range parsed.Propagations {
		config.TGo[name] = types.TGoOperation{Wrap: op.Wrap, Position: op.Position, CallName: op.CallName}
	}
	info := &types.Info{
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Defs:      make(map[*ast.Ident]types.Object),
		Uses:      make(map[*ast.Ident]types.Object),
		Scopes:    make(map[ast.Node]*types.Scope),
		Instances: make(map[*ast.Ident]types.Instance),
	}
	_, err = config.Check("example.com/p", fset, []*ast.File{parsed.File}, info)
	return info, err
}

func TestConditionalTypes(t *testing.T) {
	for _, source := range []string{
		`func f(ok bool) string { return if ok { "yes" } else { "no" } }`,
		`func f(ok bool) int { x := if ok { n:=1; n } else { 2 }; return x }`,
		`func f(ok bool) int8 { return (if ok {127} else {0})+1 }`,
		`func f(ok bool) int { return if ok { return 1 } else { 2 } }`,
		`func f(ok bool) int { return if ok { panic("stop") } else { 2 } }`,
		`func f(ok bool) *int { return if ok { nil } else { nil } }`,
		`func f(ok bool) any { return if ok { 1 } else { "two" } }`,
		`func f(ok bool) any { return -(if ok {1} else {2}) }`,
		`func f(ok bool) any { return (if ok {1} else {2}) + 3 }`,
		`func f(ok bool) any { return (if ok {1} else {2}) << 1 }`,
		`func f(ok bool) any { return !(if ok {true} else {false}) }`,
		`func f(ok bool) { var x any = if ok { 1 } else { "two" }; _ = x }`,
		`func f(ok bool) int { return if ok { if !ok {1} else {2} } else {3} }`,
		`func f(ok bool) int { return if ok {1} else if !ok {2} else {3} }`,
		`func use(x any) {} ; func f(ok bool) { use(if ok {1} else {"two"}) }`,
		`func use[T ~int](x T) T {return x}; func f[T ~int](ok bool, a,b T) T {return use(if ok {a} else {b})}`,
		`func f(ok bool) int { for { x:=if ok {break} else {1}; return x }; return 0 }`,
		`func f(ok bool) { for { if x:=if ok {break} else {1}; x>0 {} } }`,
		`func f(ok bool) { for ; if ok {break} else {true}; {} }`,
		`func f(ok bool) { for ; if ok {continue} else {true}; {} }`,
		`func f(ok bool) { for i:=0; i<3; i+=if ok {break} else {1} {} }`,
		`func f(ok bool) { for i:=0; i<3; i+=if ok {continue} else {1} {} }`,
		`func f(ok bool) { for { if x:=if ok {continue} else {1}; x>0 {break} } }`,
		`func f(ok bool) { for { _=if x:=if ok {break} else {1}; x>0 {2} else {3} } }`,
		`var ok = true; var x = if ok {1} else {2}`,
		`func value() int {return 7}; func f(ok bool) int {return if ok {goto tail; tail: value()} else {0}}`,
		`func f(ok bool) []any { return append([]any{}, if ok {1} else {"two"}) }`,
		`func f(ok bool) []*int { return append([]*int{}, if ok {nil} else {nil}) }`,
		`type item struct { value any }; func f(ok bool) item {return item{value: if ok {1} else {"two"}}}`,
		`func f(ok bool) map[string]any {return map[string]any{"key": if ok {1} else {"two"}}}`,
		`func f(ok bool) []any {return []any{if ok {1} else {"two"}}}`,
		`func f(ok bool, ch chan any) { ch <- if ok {1} else {"two"} }`,
		`func f(ok bool) int {return int(if ok {127} else {0})}`,
		`func f(ok bool) int { return len(if ok {[2]int{1,2}} else {[2]int{3,4}}) }`,
		`func f(ok bool) int { return cap(if ok {[2]int{1,2}} else {[2]int{3,4}}) }`,
		`func f(ok bool) []byte { return append([]byte{}, if ok {255} else {0}) }`,
		`func f(ok bool) int { var x int; x=if ok {goto done} else {1}; _=x; done: return 2 }`,
		`func f(ok bool) int { start: x:=if ok {goto start} else {1}; return x }`,
		`func f(ok bool) int { outer: for { x:=if ok {break outer} else {1}; return x }; return 0 }`,
		`func next() (int,error) {return 1,nil}; func f(ok bool) (int,error) { x:=if ok {next()!} else {next()!!}; return x,nil }`,
	} {
		t.Run(source, func(t *testing.T) {
			if _, err := checkTGo(t, source); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConditionalTypeErrors(t *testing.T) {
	for _, test := range []struct{ source, message string }{
		{`func f(ok bool) { x:=if ok {1} else {"two"}; _=x }`, "explicit type annotation"},
		{`func f(ok bool) { x:=if ok {nil} else {nil}; _=x }`, "explicit type annotation"},
		{`func f(ok bool) { x:=if ok {1}; _=x }`, "requires else"},
		{`func f(ok bool) { x:=if ok {} else {1}; _=x }`, "final value"},
		{`func f(ok bool) { x:=if ok {n:=1; _=n} else {1}; _=x }`, "final value"},
		{`func f(ok bool) { const x=if ok {1} else {2}; _=x }`, "not constant"},
		{`func f(ok bool) { const x=len(if ok {[2]int{1,2}} else {[2]int{3,4}}); _=x }`, "not constant"},
		{`func f(ok bool) { var x [if ok {1} else {2}]int; _=x }`, "must be constant"},
		{`func f(ok bool) { x:=if 1 {1} else {2}; _=x }`, "Boolean"},
		{`func f(ok bool) int8 {return if ok {128} else {0}}`, "overflows"},
		{`func pair() (int,int){return 1,2}; func f(ok bool) {x:=if ok {pair()} else {1}; _=x}`, "multiple-value"},
		{`func f(ok bool) { x:=if ok {n:=1; n} else {n}; _=x }`, "undefined: n"},
		{`func f(ok bool) { x:=if ok {goto missing} else {1}; _=x }`, "label missing not declared"},
		{`func f(ok bool) { goto inner; x:=if ok {inner: return} else {1}; _=x }`, "jumps into block"},
		{`func f(ok bool) { x:=if ok {goto done} else {1}; y:=2; done: _,_=x,y }`, "jumps over variable declaration"},
		{`func f(ok bool) int { for { _=if ok {break} else {1} } }`, "missing return"},
		{`func f(ok bool) int { for i:=0;;i+=if ok {break} else {1} {} }`, "missing return"},
		{`func f(ok bool) int { outer: for { _=if ok {break outer} else {1} } }`, "missing return"},
		{`func f(ok bool) { if x:=if ok {goto missing} else {1}; x>0 {} }`, "label missing not declared"},
		{`var x = if true {return 1} else {2}`, "enclosing function"},
		{`func f() {}; var x = if true {defer f(); 1} else {2}`, "enclosing function"},
		{`var x = if true {unused:=1; 2} else {3}`, "declared and not used"},
		{`func f(ok bool) int { return if ok {loop: for {break loop}} else {1} }`, "final value"},
	} {
		t.Run(test.source, func(t *testing.T) {
			_, err := checkTGo(t, test.source)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error: %v, want %q", err, test.message)
			}
		})
	}
}
