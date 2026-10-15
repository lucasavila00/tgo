package types_test

import (
	"go/token"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

// TestNativeLabelScopes checks jumps across expression scopes and source declarations.
func TestNativeLabelScopes(t *testing.T) {
	for _, source := range []string{
		`func f(){goto done; _=if x:=1; true {x} else {x}; done:return}`,
		`func f(){goto done; _=switch x:=1;x {case 1:x;default:x}; done:return}`,
		`func f(){goto done; _=switch x:=any(1).(type) {case int:x;default:0}; done:return}`,
		`func f(){goto done; _=[]int{x for x:=range 3}; done:return}`,
		`func f(){start: _=if x:=1;true {goto start} else {x}}`,
		`func f(){_=if x:=1;true {goto done} else {x}; done:return}`,
		`func f(){_=if true {start: x:=1; if x>0 {goto start}; x} else {0}}`,
	} {
		t.Run(source, func(t *testing.T) {
			if _, err := checkTGo(t, source); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, test := range []struct{ source, message string }{
		{`func f(){goto inner; _=if x:=1;true {inner:x} else {x}}`, "jumps into block"},
		{`func f(){goto done; x:=if true {1} else {2}; done:_=x}`, "jumps over variable declaration"},
		{`func f(){_=if true {goto done; x:=1; done:x} else {0}}`, "jumps over variable declaration"},
		{`func f(){_=if true {func(){goto done}();1} else {0}; done:return}`, "label done not declared"},
	} {
		t.Run(test.source, func(t *testing.T) {
			_, err := checkTGo(t, test.source)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error: %v; want %q", err, test.message)
			}
		})
	}
}

// TestNativeArrayRuntimeFacts checks len and cap classification after nested range headers.
func TestNativeArrayRuntimeFacts(t *testing.T) {
	info, err := checkTGo(t, `type A [2]int
		func f(flag bool) int {
			counter:=0
			n:=len(if flag {counter++;for range [0]int{} {}; A{counter}} else {A{}})
			c:=cap(switch flag {case true:counter++;for range [0]int{} {}; A{counter};default:A{}})
			m:=len([]A{A{} for range [0]int{}}[0])
			const plainLen=len([3]int{})
			const plainCap=cap((*[4]int)(nil))
			return n+c+m+plainLen+plainCap+counter
		}`)
	if err != nil {
		t.Fatal(err)
	}
	var runtime, constant int
	for expression, value := range info.Types {
		call, ok := expression.(*ast.CallExpr)
		if !ok {
			continue
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok || (name.Name != "len" && name.Name != "cap") {
			continue
		}
		if value.Type != types.Typ[types.Int] {
			t.Fatalf("%s has type %v", name.Name, value.Type)
		}
		if value.Value == nil {
			runtime++
		} else {
			constant++
		}
	}
	if runtime != 3 || constant != 2 {
		t.Fatalf("runtime calls=%d; constant calls=%d; want 3 and 2", runtime, constant)
	}
}

// TestNativeArrayConstantErrors rejects a constant that would skip native statements.
func TestNativeArrayConstantErrors(t *testing.T) {
	for _, source := range []string{
		`func f(flag bool){const n=len(if flag {for range [0]int{} {}; [2]int{}} else {[2]int{}});_=n}`,
		`func f(flag bool){const n=cap(switch flag {case true:for range [0]int{} {};[2]int{};default:[2]int{}});_=n}`,
		`func f(){const n=len([][2]int{[2]int{} for range [0]int{}}[0]);_=n}`,
		`func f(flag bool)int{const n=len(if flag {return 1} else {for range [0]int{} {};[2]int{}});return n}`,
	} {
		t.Run(source, func(t *testing.T) {
			_, err := checkTGo(t, source)
			if err == nil || !strings.Contains(err.Error(), "not constant") {
				t.Fatalf("error: %v; want a nonconstant initializer", err)
			}
		})
	}
}

// TestNativePackageClosureUsage checks captured locals after all nested closure bodies.
func TestNativePackageClosureUsage(t *testing.T) {
	for _, source := range []string{
		`var f=if x:=1;true {func()int{return x}} else {func()int{return x}}`,
		`var f=switch x:=1;x {case 1:func()int{return x};default:func()int{return x}}`,
		`var f=switch x:=any(1).(type) {case int:func()int{return x};default:func()int{return 0}}`,
		`func f(v any)func()int{return switch x:=v.(type) {case int:func()int{return x};default:func()int{return 0}}}`,
		`var fs=[]func()func()int{func()func()int{return func()int{return x}} for x:=range 3}`,
		`var f=if true {x:=1;func()func()int{return func()int{return x}}} else {func()func()int{return nil}}`,
		`var f=switch 1 {case 1:x:=1;func()int{return x};default:func()int{return 0}}`,
	} {
		t.Run(source, func(t *testing.T) {
			if _, err := checkTGo(t, source); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, source := range []string{
		`var f=if x:=1;true {func()int{return 0}} else {func()int{return 0}}`,
		`var f=switch x:=1;true {case true:func()int{return 0};default:func()int{return 0}}`,
		`var f=switch x:=any(1).(type) {case int:func()int{return 0};default:func()int{return 0}}`,
		`func f(v any)func()int{return switch x:=v.(type) {case int:func()int{return 0};default:func()int{return 0}}}`,
		`var fs=[]func()int{func()int{return 0} for x:=range 3}`,
	} {
		t.Run(source, func(t *testing.T) {
			var diagnostics []error
			fset := token.NewFileSet()
			parsed, err := frontend.ParseFile(fset, "main.tgo", "main.tgo", []byte("package p\n"+source))
			if err != nil {
				t.Fatal(err)
			}
			config := types.Config{Error: func(err error) { diagnostics = append(diagnostics, err) }}
			_, err = config.Check("example.com/p", fset, []*ast.File{parsed.File}, nil)
			if err == nil || len(diagnostics) != 1 || !strings.Contains(err.Error(), "declared and not used") {
				t.Fatalf("error: %v; diagnostics: %v; want one unused binding", err, diagnostics)
			}
		})
	}
}

// TestPropagationTuplePlacement checks ordinary Go rules for a call with several success results.
func TestPropagationTuplePlacement(t *testing.T) {
	prefix := `func pair()(int,int,error){return 1,2,nil};func use(a,b int){};func many(xs...int){}`
	for _, source := range []string{
		`func f()error{use(pair()!);return nil}`,
		`func f()error{many(pair()!);return nil}`,
		`func f()(int,int,error){a,b:=pair()!;return a,b,nil}`,
	} {
		if _, err := checkTGo(t, prefix+";"+source); err != nil {
			t.Fatal(err)
		}
	}
	for _, source := range []string{
		`func f()error{use(pair()!,1);return nil}`,
		`func f()(int,int,error){return pair()!,nil}`,
		`func f()error{many(pair()!...);return nil}`,
	} {
		if _, err := checkTGo(t, prefix+";"+source); err == nil {
			t.Fatalf("accepted an invalid tuple placement: %s", source)
		}
	}
}

// TestGenericNativeResultFacts checks branch result types after generic argument inference.
func TestGenericNativeResultFacts(t *testing.T) {
	info, err := checkTGo(t, `func pick[T any](value T)T{return value}
		func f(flag bool)[2]int{
			a:=pick(if flag {for range [0]int{} {};[2]int{}} else {[2]int{}})
			b:=pick(switch flag {case true:for range [0]int{} {};[2]int{};default:[2]int{}})
			return [2]int{a[0],b[0]}
		}`)
	if err != nil {
		t.Fatal(err)
	}
	var branches int
	for expression, value := range info.Types {
		switch expression.(type) {
		case *ast.IfExpr, *ast.SwitchExpr:
			if value.Type.String() != "[2]int" || value.Value != nil {
				t.Fatalf("branch result: %+v; want a runtime [2]int", value)
			}
			branches++
		}
	}
	if branches != 2 || len(info.Instances) != 2 {
		t.Fatalf("branches=%d; generic instances=%d; want 2 each", branches, len(info.Instances))
	}
}
