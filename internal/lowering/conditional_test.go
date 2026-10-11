package lowering

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lucasavila00/tgo/internal/checker"
	"golang.org/x/tools/go/packages"
)

func TestConditionalExecution(t *testing.T) {
	input := `package main
import "fmt"
func effect(label string, value int) int { fmt.Print(label); return value }
func consume(a,b,c int) { fmt.Printf("=%d;", a+b+c) }
func choose(ok bool) string {
 return if ok { fmt.Print("Y"); name:="yes"; name } else { fmt.Print("N"); "no" }
}
func early(ok bool) int { return if ok {return 7} else {3} }
func small(ok bool) int8 { return (if ok {127} else {0})+1 }
func nested(ok bool) int { return if ok { if !ok {1} else {2} } else if !ok {3} else {4} }
func deferred() int { return if true {defer fmt.Print("D"); 5} else {0} }
func movedBinding(skip bool) {
 i:=4
 var values []func() int
 loop: for i:=if skip {break loop} else {i}; i<6; i++ {
  values=append(values,func()int{return i})
 }
 if i!=4 {panic("outer binding")}
 if !skip && (len(values)!=2 || values[0]()!=4 || values[1]()!=5) {panic("iteration bindings")}
}
func sharedExit(ok bool) int {
 var ignored int = if ok {return 7} else {return 3}
 _=ignored
}
func main() {
 if sharedExit(true)!=7 || sharedExit(false)!=3 {panic("shared exit")}
 movedBinding(false)
 movedBinding(true)
 fmt.Print(choose(true), ";", choose(false), ";")
 consume(effect("L",1), if true {effect("B",2)} else {effect("X",0)}, effect("R",3))
 _=false && if true {fmt.Print("X"); true} else {false}
 fmt.Print(early(true), ";", early(false), ";", small(true), ";", small(false), ";", nested(true), ";", nested(false), ";", deferred())
}
`
	dir := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example.com/conditional\n\ngo 1.27.1\n", "main_tgo.go": input} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	roots, err := packages.Load(&packages.Config{Dir: dir, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps | packages.NeedExportFile | packages.NeedModule}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 {
		t.Fatalf("packages: %d", len(roots))
	}
	checked, err := checker.NewLoader(dir, roots, map[string]string{"main_tgo.go": "main.tgo"}, make(map[string]bool)).Check(roots[0])
	if err != nil {
		t.Fatal(err)
	}
	source, err := SourceFromCheckedPackage(checked, dir)
	if err != nil {
		t.Fatal(err)
	}
	first, err := source.LowerPropagations(nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := source.LowerPropagations(nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range first {
		if !bytes.Equal(content, second[name]) {
			t.Fatalf("output changed between calls: %s", name)
		}
		if err := os.WriteFile(name, content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command("go", "run", ".")
	command.Dir = dir
	actual, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s\n%s", err, actual, first[filepath.Join(dir, "main_tgo.go")])
	}
	if string(actual) != "YNyes;no;LBR=6;D7;3;-128;1;2;3;5" {
		t.Fatalf("output: %q", actual)
	}
}
