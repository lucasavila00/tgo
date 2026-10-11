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
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/program\n\ngo 1.27.1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.tgo"), []byte(input), 0600); err != nil {
		t.Fatal(err)
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
