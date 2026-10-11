package compilerv2

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPropagationCalls(t *testing.T) {
	input := `package main
import("fmt";"errors")
var sentinel=errors.New("failure")
func effect(name string)int{fmt.Print(name);return 1}
func target(fail bool)(int,error){fmt.Print("T");if fail{return 0,sentinel};return 2,nil}
func consume(a,b,c int){fmt.Print(a,b,c)}
func run(fail bool)error{consume(effect("L"),target(fail)!!,effect("R"));return nil}
func wrapped()error{target(true)!;return nil}
func main(){fmt.Print(run(false)==nil,"|");fmt.Print(run(true)==sentinel,"|");e:=wrapped();fmt.Print(e.Error(),"|",errors.Is(e,sentinel))}
`
	if actual := runPropagation(t, input); actual != "LTR1 2 1true|LTtrue|Ttarget: failure|true" {
		t.Fatalf("got %q", actual)
	}
}

func runPropagation(t *testing.T, input string) string {
	return runPropagationFiles(t, map[string]string{"main.tgo": input})
}

func runPropagationFiles(t *testing.T, sources map[string]string) string {
	t.Helper()
	defer func() {
		if failure := recover(); failure != nil {
			t.Fatalf("compile: %v", failure)
		}
	}()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/program\n\ngo 1.27.1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for name, input := range sources {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
	}
	files := Compile(dir)
	output := t.TempDir()
	for name, content := range files {
		destination := filepath.Join(output, name)
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, content, 0600); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, ".go") {
			t.Logf("%s:\n%s", name, content)
		}
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = output
	actual, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run generated source: %v\n%s", err, actual)
	}
	for name := range sources {
		if strings.HasSuffix(name, "_test.tgo") {
			command := exec.Command("go", "test", "./...")
			command.Dir = output
			result, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("generated tests: %v\n%s", err, result)
			}
			break
		}
	}
	return string(actual)
}

func TestPropagationContexts(t *testing.T) {
	for _, test := range []struct{ name, body, want string }{
		{"assignment", `a:=[2]int{};a[step("I")!!]=step("V")!!;fmt.Print(a);return nil`, "IV[0 1]S|IE|IVE|"},
		{"declaration", `a:=step("A")!!;b:=step("B")!!;fmt.Print(a,b);return nil`, "AB1 1S|AE|ABE|"},
		{"short circuit", `fmt.Print(false && predicate("B")!!);fmt.Print(true && predicate("C")!!);return nil`, "falseCtrueS|falseCE|falseCtrueS|"},
		{"loop header", `for i:=step("I")!!;i<step("C")!!+1;i++{fmt.Print("b")};return nil`, "ICbCS|IE|ICE|"},
		{"loop post", `for i:=0;i<2;i+=step("P")!!{fmt.Print("b");continue};return nil`, "bPbPS|bPE|bPbPE|"},
		{"switch", `switch step("T")!!{case step("C")!!:fmt.Print("m");fallthrough;case 2:fmt.Print("f")};return nil`, "TCmfS|TE|TCE|"},
		{"type switch", `switch any(step("T")!!).(type){case int:fmt.Print("m")};return nil`, "TmS|TE|TmS|"},
		{"range target", `a:=[2]int{};for a[step("I")!!]=range []int{1,2}{fmt.Print("b")};return nil`, "IbIbS|IE|IbIE|"},
		{"select target", `ch:=make(chan int,1);ch<-1;a:=[2]int{};select{case a[step("I")!!]=<-ch:fmt.Print("b")};return nil`, "IbS|IE|IbS|"},
		{"select send", `ch:=make(chan int,1);select{case ch<-step("V")!!:fmt.Print("b")};return nil`, "VbS|VE|VbS|"},
		{"defer argument", `defer fmt.Print(step("D")!!);step("B")!!;return nil`, "DB1S|DE|DB1E|"},
		{"tuple", `a,b:=pair()!!;fmt.Print(a,b);return nil`, "P1xS|PE|P1xS|"},
		{"error only", `only()!!;fmt.Print("b");return nil`, "ObS|OE|ObS|"},
		{"nested propagation", `fmt.Print(identity(step("I")!!)!!);return nil`, "IF1S|IE|IFE|"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := `package main
import("fmt";"errors")
var sentinel=errors.New("failure")
var visit,failAt int
func step(name string)(int,error){fmt.Print(name);visit++;if visit==failAt{return 0,sentinel};return 1,nil}
func predicate(name string)(bool,error){_,err:=step(name);return true,err}
func pair()(int,string,error){v,err:=step("P");return v,"x",err}
func only()error{_,err:=step("O");return err}
func identity[T any](v T)(T,error){_,err:=step("F");return v,err}
func run()error{` + test.body + `}
func main(){for _,failure:=range []int{0,1,2}{visit=0;failAt=failure;err:=run();if err==nil{fmt.Print("S|")}else if err==sentinel{fmt.Print("E|")}else{panic(err)}}}
`
			if actual := runPropagation(t, input); actual != test.want {
				t.Fatalf("got %q, want %q", actual, test.want)
			}
		})
	}
}

func TestPropagationReturns(t *testing.T) {
	input := `package main
import("fmt";"errors")
var sentinel=errors.New("failure")
var fail bool
type Value int
func get()(Value,error){if fail{return 7,sentinel};return 8,nil}
func unnamed(Value string)(Value,error){return get()!!,nil}
func named()(value Value,err error){value=5;err=sentinel;get()!!;return}
func generic[T any](value T)(T,error){get()!!;return value,nil}
func closure()error{call:=func()(Value,error){return get()!!,nil};v,e:=call();fmt.Print(v,e==sentinel);return nil}
func main(){for _,failure:=range []bool{false,true}{fail=failure;v,e:=unnamed("shadow");fmt.Print(v,e==sentinel,"|");v,e=named();fmt.Print(v,e==sentinel,"|");s,e:=generic("value");fmt.Print(s,e==sentinel,"|");closure();fmt.Print("|")}}
`
	if actual := runPropagation(t, input); actual != "8 false|5 true|valuefalse|8 false|0 true|0 true|true|0 true|" {
		t.Fatalf("got %q", actual)
	}
}

func TestPropagationControl(t *testing.T) {
	for _, test := range []struct{ name, body, want string }{
		{"go argument", `done:=make(chan struct{});go func(v int){fmt.Print("g");close(done)}(step("G")!!);<-done;return nil`, "GgS|GE|GgS|"},
		{"label", `i:=0;again:for i<2{i++;step("B")!!;if i==1{continue again};break again};return nil`, "BBS|BE|BBE|"},
		{"goto", `i:=0;again:i++;step("B")!!;if i<2{goto again};return nil`, "BBS|BE|BBE|"},
		{"recover", `defer func(){if recover()!=nil{fmt.Print("r")}}();panic(step("P")!!);`, "PrS|PE|PrS|"},
		{"ordered stores", `a:=[1]int{};a[step("I")!!],a[0]=step("V")!!,step("W")!!;return nil`, "IVWrS|IE|IVE|"},
		{"literal context", `type S struct{v int};a:=[]S{{step("A")!!},{step("B")!!}};fmt.Print(a);return nil`, "AB[{1} {1}]S|AE|ABE|"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := test.body
			if test.name == "ordered stores" {
				body = `defer func(){if recover()!=nil{fmt.Print("r")}}();` + body
			}
			input := `package main
import("fmt";"errors")
var sentinel=errors.New("failure")
var visit,failAt int
func step(name string)(int,error){fmt.Print(name);visit++;if visit==failAt{return 0,sentinel};return 1,nil}
func run()error{` + body + `}
func main(){for _,failure:=range []int{0,1,2}{visit=0;failAt=failure;err:=run();if err==nil{fmt.Print("S|")}else if err==sentinel{fmt.Print("E|")}else{panic(err)}}}
`
			if actual := runPropagation(t, input); actual != test.want {
				t.Fatalf("got %q, want %q", actual, test.want)
			}
		})
	}
}

func TestPropagationImports(t *testing.T) {
	foreign := `package foreign
import("fmt";"errors")
var Failure=errors.New("foreign")
var visit,failAt int
type value int
type flag bool
type item struct{Number int}
type key struct{ID int}
type List []item
type Table map[key]item
type Generic[K comparable] map[K]item
func Reset(failure int){visit=0;failAt=failure}
func Number(name string)(value,error){fmt.Print(name);visit++;if visit==failAt{return 0,Failure};return 1,nil}
func Boolean()(flag,error){_,err:=Number("C");return true,err}
func Factory()(func()(value,error),error){_,err:=Number("F");return func()(value,error){return Number("V")},err}
`
	for _, test := range []struct{ name, body, want string }{
		{"private value", `v:=foreign.Number("A")!!;fmt.Printf("%T",v);return nil`, "Aforeign.valueS|AE|Aforeign.valueS|"},
		{"private Boolean", `v:=true && foreign.Boolean()!!;fmt.Printf("%T",v);return nil`, "Cforeign.flagS|CE|Cforeign.flagS|"},
		{"private literal", `v:=foreign.List{{Number:int(foreign.Number("A")!!)},{Number:int(foreign.Number("B")!!)}};fmt.Print(v);return nil`, "AB[{1} {1}]S|AE|ABE|"},
		{"private map", `v:=foreign.Table{{ID:1}:{Number:int(foreign.Number("A")!!)},{ID:2}:{Number:int(foreign.Number("B")!!)}};fmt.Print(len(v));return nil`, "AB2S|AE|ABE|"},
		{"function value", `call:=foreign.Number;call("A")!;return nil`, "AS|Acall: foreignE|AS|"},
		{"returned function", `foreign.Factory()!!()!;return nil`, "FVS|FE|FVforeign.Factory(): foreignE|"},
	} {
		t.Run(test.name, func(t *testing.T) {
			main := `package main
import("fmt";"errors";"example.com/program/foreign")
func run()error{` + test.body + `}
func main(){for _,failure:=range []int{0,1,2}{foreign.Reset(failure);err:=run();if err==nil{fmt.Print("S|")}else if errors.Is(err,foreign.Failure){if err!=foreign.Failure{fmt.Print(err.Error())};fmt.Print("E|")}else{panic(err)}}}
`
			if actual := runPropagationFiles(t, map[string]string{"main.tgo": main, "foreign/foreign.go": foreign}); actual != test.want {
				t.Fatalf("got %q, want %q", actual, test.want)
			}
		})
	}
	t.Run("generic map key", func(t *testing.T) {
		main := `package main
import("fmt";"example.com/program/foreign")
func run[K comparable](key K)error{v:=foreign.Generic[K]{key:{Number:int(foreign.Number("A")!!)},key:{Number:int(foreign.Number("B")!!)}};fmt.Print(len(v));return nil}
func main(){for _,failure:=range []int{0,1,2}{foreign.Reset(failure);err:=run(7);if err==nil{fmt.Print("S|")}else if err==foreign.Failure{fmt.Print("E|")}else{panic(err)}}}
`
		if actual := runPropagationFiles(t, map[string]string{"main.tgo": main, "foreign/foreign.go": foreign}); actual != "AB1S|AE|ABE|" {
			t.Fatalf("got %q", actual)
		}
	})
}

func TestPropagationIntegerRange(t *testing.T) {
	for _, test := range []struct{ name, body, want string }{
		{"named integer", `a:=[]N{9,9};for a[step("I")!!]=range 2{fmt.Print(a[1])};return nil`, "I0I1S|IE|I0IE|"},
		{"zero iterations", `var a *[2]N;for a[step("I")!!]=range 0{};return nil`, "S|S|S|"},
		{"negative iterations", `a:=[]int{};for a[step("I")!!]=range -1{};return nil`, "S|S|S|"},
		{"nil array", `var a *[2]N;for a[step("I")!!]=range 2{};return nil`, "IrS|IE|IrS|"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := `package main
import("fmt";"errors")
var sentinel=errors.New("failure")
var visit,failAt int
type N uint64
func step(name string)(int,error){fmt.Print(name);visit++;if visit==failAt{return 0,sentinel};return 1,nil}
func run()error{defer func(){if recover()!=nil{fmt.Print("r")}}();` + test.body + `}
func main(){for _,failure:=range []int{0,1,2}{visit=0;failAt=failure;err:=run();if err==nil{fmt.Print("S|")}else if err==sentinel{fmt.Print("E|")}else{panic(err)}}}
`
			if actual := runPropagation(t, input); actual != test.want {
				t.Fatalf("got %q, want %q", actual, test.want)
			}
		})
	}
}

func TestPropagationGenericStorage(t *testing.T) {
	input := `package main
import("fmt";"errors")
var sentinel=errors.New("failure")
var visit,failAt int
func step(name string)(int,error){fmt.Print(name);visit++;if visit==failAt{return 0,sentinel};return 1,nil}
func set[T interface{~[2]int|~[]int}](a,b T)error{
 index:=func()(int,error){a=b;return step("I")}
 {T:=3;_ =T;a[index()!!]=step("V")!!}
 fmt.Print(a);return nil
}
func arrays[T interface{~[2]int|~[3]int}](a,b T)error{index:=func()(int,error){a=b;return step("I")};a[index()!!]=step("V")!!;fmt.Print(a);return nil}
type Box[T interface{~[2]int|~[]int}] struct{Value T}
func(b *Box[T]) Set()error{defer func(){if recover()!=nil{fmt.Print("r")}}();b.Value[step("I")!!]=step("V")!!;fmt.Print(b.Value);return nil}
func show(err error){if err==nil{fmt.Print("S|")}else if err==sentinel{fmt.Print("E|")}else{panic(err)}}
func main(){for _,failure:=range []int{0,1,2}{
 visit=0;failAt=failure;show(set([2]int{},[2]int{7,8}))
 visit=0;failAt=failure;a:=[]int{0,0};show(set(a,[]int{7,8}));fmt.Print(a,"|")
 visit=0;failAt=failure;show(arrays([2]int{},[2]int{7,8}))
 visit=0;failAt=failure;var b *Box[[2]int];show(b.Set())
}}
`
	want := "IV[7 1]S|IV[7 8]S|[0 1]|IV[7 1]S|IVrS|IE|IE|[0 0]|IE|IE|IVE|IVE|[0 0]|IVE|IVE|"
	if actual := runPropagation(t, input); actual != want {
		t.Fatalf("got %q, want %q", actual, want)
	}
}

func TestPropagationErrorValues(t *testing.T) {
	input := `package main
import("fmt";"errors")
type Failure struct{}
func(*Failure)Error()string{return "failure"}
var sentinel=&Failure{}
var any=7
func target(typedNil bool)error{if typedNil{var p *Failure;return p};return sentinel}
func direct()error{target(true)!!;panic("not reached")}
func wrapped()error{target(false)!;panic("not reached")}
func shadow(nil,true,false int)error{fmt.Print(nil,true,false);target(0!=0)!!;var success error;return success}
func main(){var p *Failure;e:=direct();fmt.Print(e==p,"|");e=wrapped();var caught *Failure;fmt.Print(e.Error(),"|",errors.Is(e,sentinel),"|",errors.As(e,&caught),caught==sentinel,"|");fmt.Print(shadow(0,1,2)==sentinel)}
`
	if actual := runPropagation(t, input); actual != "true|target: failure|true|true true|0 1 2true" {
		t.Fatalf("got %q", actual)
	}
}

func TestPropagationProject(t *testing.T) {
	sources := map[string]string{
		"main.tgo": `package main
import("fmt";"embed";"example.com/program/library")
//go:embed message.txt
var message string
var _ embed.FS
func run()error{v:=library.Read()!!;fmt.Print(v,message);return nil}
func main(){if err:=run();err!=nil{panic(err)}}
`,
		"message.txt": "asset",
		"library/value.tgo": `package library
func Read()(int,error){return 3,nil}
`,
		"ignored.tgo": "//go:build excluded\n\npackage main\nfunc invalid(){this is not Go!!}\n",
		"value_test.tgo": `package main
import("testing";"example.com/program/library")
func testRead()(int,error){return library.Read()!!,nil}
func TestRead(t *testing.T){v,err:=testRead();if v!=3 || err!=nil{t.Fatal(v,err)}}
`,
	}
	if actual := runPropagationFiles(t, sources); actual != "3asset" {
		t.Fatalf("got %q", actual)
	}
}
