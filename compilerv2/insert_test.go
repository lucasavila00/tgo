package compilerv2

import (
	"bytes"
	"go/ast"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInsertReturn(t *testing.T) {
	cases := []struct{ name, code, target, want string }{
		{"call", `func run(err error) error { use(left(),right()); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "right()", "LRtrue"},
		{"short circuit reached", `func run(err error) error { useBool(yes() && yes()); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "yes() && yes()", "YYtrue"},
		{"short circuit skipped", `func run(err error) error { useBool(no() && yes()); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "yes()", "NBfalse"},
		{"short circuit operand", `func run(err error) error { useBool(yes() && no()); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "no()", "YNtrue"},
		{"closure", `func run(err error) error { fn:=func() error { use(left(),right()); return nil }; _=fn(); fmt.Print("O"); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "right()", "LROfalse"},
		{"tuple", `func run(err error) error { a,b:=pair(); fmt.Print(a,b); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "pair()", "Ptrue"},
		{"loop condition", `func run(err error) error { for i:=0; check(i); i++ { fmt.Print(i) }; return nil }; func main() { fmt.Print(run(failure)==failure) }`, "check(i)", "Ctrue"},
		{"loop post", `func run(err error) error { for i:=0; i<2; i=next(i) { fmt.Print(i); continue }; return nil }; func main() { fmt.Print(run(failure)==failure) }`, "next(i)", "0Ttrue"},
		{"defer", `func run(err error) error { defer fmt.Print("D"); use(left(),right()); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "right()", "LRDtrue"},
		{"assignment target", `func run(err error) error { a:=[3]int{}; a[left()]=right(); fmt.Print(a); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "right()", "LRtrue"},
		{"assignment index", `func run(err error) error { a:=[3]int{}; a[left()]=right(); fmt.Print(a); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "left()", "Ltrue"},
		{"map target", `func run(err error) error { a:=map[int]int{}; a[left()]=right(); fmt.Print(a); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "right()", "LRtrue"},
		{"composite", `func run(err error) error { a:=[]int{left(),right()}; fmt.Print(a); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "right()", "LRtrue"},
		{"switch case", `func run(err error) error { switch 2 { case left(): fmt.Print("A"); case right(): fmt.Print("B"); default: fmt.Print("X") }; return nil }; func main() { fmt.Print(run(failure)==failure) }`, "right()", "LRtrue"},
		{"switch skipped case", `func run(err error) error { switch 1 { case left(): fmt.Print("A"); case right(): fmt.Print("B") }; return nil }; func main() { fmt.Print(run(failure)==failure) }`, "right()", "LAfalse"},
		{"range expression", `func values() []int { fmt.Print("V"); return []int{1,2} }; func run(err error) error { for _,v:=range values() { fmt.Print(v) }; return nil }; func main() { fmt.Print(run(failure)==failure) }`, "values()", "Vtrue"},
		{"select receive", `func run(err error) error { c:=make(chan int,1); c<-1; select { case v:= <-c: fmt.Print(v) }; return nil }; func main() { fmt.Print(run(failure)==failure) }`, "<-c", "true"},
		{"select target", `func run(err error) error { c:=make(chan int,1); c<-1; a:=[3]int{}; select { case a[left()]= <-c: fmt.Print(a) }; return nil }; func main() { fmt.Print(run(failure)==failure) }`, "left()", "Ltrue"},
		{"comma map", `func run(err error) error { a:=map[int]int{1:2}; v,ok:=a[1]; fmt.Print(v,ok); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "a[1]", "true"},
		{"comma assertion", `func run(err error) error { var a any=1; v,ok:=a.(int); fmt.Print(v,ok); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "a.(int)", "true"},
		{"comma receive", `func run(err error) error { c:=make(chan int,1); c<-1; v,ok:= <-c; fmt.Print(v,ok); return nil }; func main() { fmt.Print(run(failure)==failure) }`, "<-c", "true"},
		{"type switch operand", `func interfaceValue() any { fmt.Print("I"); return 1 }; func run(err error) error { switch v:=interfaceValue().(type) { case int:fmt.Print(v) }; return nil }; func main() { fmt.Print(run(failure)==failure) }`, "interfaceValue()", "Itrue"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := `package main
import "fmt"
import "errors"
var failure=errors.New("failure")
func left() int { fmt.Print("L"); return 1 }
func right() int { fmt.Print("R"); return 2 }
func use(a,b int) { fmt.Print("U") }
func useBool(v bool) { fmt.Print("B") }
func yes() bool { fmt.Print("Y"); return true }
func no() bool { fmt.Print("N"); return false }
func pair() (int,int) { fmt.Print("P"); return 1,2 }
func check(i int) bool { fmt.Print("C"); return i<2 }
func next(i int) int { fmt.Print("T"); return i+1 }
` + tc.code
			actual := runInsertion(t, input, tc.target, func([]ast.Expr) []ast.Stmt {
				return []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}}}
			})
			if actual != tc.want {
				t.Fatalf("got %q, want %q", actual, tc.want)
			}
		})
	}
}

func runInsertion(t *testing.T, input, target string, after After) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example.com/insertion\n\ngo 1.27.1\n", "main.go": input} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, err := Load(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	var site Site
	for _, candidate := range source.Sites {
		if input[candidate.Start:candidate.End] == target {
			site = candidate
		}
	}
	if site.Kind == "" {
		t.Fatalf("missing site %s", target)
	}
	var original bytes.Buffer
	if err := format.Node(&original, source.Package.Fset, source.Package.Syntax[0]); err != nil {
		t.Fatal(err)
	}
	files, err := source.InsertAfter(site, after)
	if err != nil {
		t.Fatal(err)
	}
	var unchanged bytes.Buffer
	if err := format.Node(&unchanged, source.Package.Fset, source.Package.Syntax[0]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original.Bytes(), unchanged.Bytes()) {
		t.Fatal("original AST changed")
	}
	for name, content := range files {
		if err := os.WriteFile(name, content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run generated source: %v\n%s\n%s", err, output, files[filepath.Join(dir, "main.go")])
	}
	return string(output)
}
