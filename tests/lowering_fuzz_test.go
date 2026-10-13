//go:build loweringfuzz

package tests

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

// Each body uses P for propagation. The reference uses a separate Go function
// that stops run on an error. It does not use the compiler's lowering rules.
var fuzzBodies = []string{
	`emit(read(0), P(1), read(2))`,
	`emit(readX() + P(1))`,
	`emit(P(1) + readX())`,
	`emit(at(P(1)), read(2))`,
	`emit(at(read(0)), P(1))`,
	`a[read(0)] = P(1); emit(a[:]...)`,
	`s[read(0)] = P(1); emit(s...)`,
	`m[read(0)] = P(1); emit(m[0], m[1], m[2], m[3])`,
	`a[read(0)] += P(1); emit(a[:]...)`,
	`s[read(0)] += P(1); emit(s...)`,
	`m[read(0)] += P(1); emit(m[0], m[1], m[2], m[3])`,
	`i := 0; i, a[i] = P(1), P(2); emit(i); emit(a[:]...)`,
	`emit(len(s[P(1):P(2)]))`,
	`emit(len(a[P(1):P(2):P(3)]))`,
	`emit(append(s, read(0), P(1), read(2))...)`,
	`emit([]int{read(0), P(1), read(2)}...)`,
	`v := map[int]int{read(0): P(1), P(2): read(3)}; emit(v[0], v[1], v[2], v[3])`,
	`if truth(read(0)) && truth(P(1)) { emit(P(2)) }`,
	`if truth(read(0)) || truth(P(1)) { emit(P(2)) }`,
	`if v := P(1); v < 2 { emit(P(2)) } else { emit(P(3)) }`,
	`switch P(1) { case read(0): emit(0); case P(2), P(3): emit(1); default: emit(2) }`,
	`switch { case truth(P(1)): emit(0); case truth(P(2)): emit(1); default: emit(2) }`,
	`for i := 0; i < 2; i++ { emit(P(1)); continue }`,
	`for i := 0; truth(P(1)) && i < 2; i++ { emit(i) }`,
	`for i := 0; i < 2; i = P(1) + 2 { emit(i); continue }`,
	`for _, v := range []int{P(1), P(2)} { emit(v) }`,
	`defer emit(read(0), P(1), read(2)); emit(9)`,
	`ch := make(chan int, 1); ch <- 5; select { case v := <-channel(P(1), ch): emit(v); default: emit(9) }`,
	`ch := make(chan int, 1); select { case ch <- P(1): emit(<-ch); default: emit(9) }`,
	`var p *int; *p = P(1)`,
	`emit(at(4), P(1))`,
	`var v any = "text"; emit(assertInt(v), P(1))`,
	`f := choose(read(0)); f(P(1), read(2))`,
	`obj := item{value: 10}; obj.add(P(1)); emit(obj.value)`,
	`obj := item{value: 10}; emit(obj.get(), P(1)); emit(obj.value)`,
	`emit(read(0), plus(read(1), P(2)), read(3))`,
	`u, v := Q(1); emit(u, v)`,
	`old := s; sliceBase()[sliceIndex()] = P(1); emit(old...); emit(s...)`,
	`old := m; mapBase()[mapIndex()] = P(1); emit(old[0], m[0])`,
	`f := receiver().add; f(P(1)); emit(x)`,
	`*pointer(), a[0] = P(1), P(2); emit(x); emit(a[:]...)`,
	`L: for i := 0; i < 2; i++ { if truth(P(1)) { continue L }; emit(i) }`,
}

var fuzzPropagation = regexp.MustCompile(`([PQ])\(([0-9]+)\)`)

func FuzzPropagationBehavior(f *testing.F) {
	for body := range fuzzBodies {
		// No error, an early error, and a later error for each context.
		for _, failure := range []byte{255, 1, 2} {
			for _, operator := range []byte{0, 1} {
				f.Add([]byte{byte(body), failure, operator, 0, 1, 2, 3})
			}
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 3 {
			return
		}
		body := fuzzBodies[int(data[0])%len(fuzzBodies)]
		wrapped := data[2]&1 != 0
		input, reference := fuzzPrograms(t, body, data, wrapped)
		checkFuzzBehavior(t, "example.com/fuzz", input, reference)
	})
}

// checkFuzzBehavior builds each program in its own module and compares its trace.
func checkFuzzBehavior(t *testing.T, moduleName, input, reference string) {
	t.Helper()
	module := []byte("module " + moduleName + "\n\ngo " + strings.TrimPrefix(runtime.Version(), "go") + "\n")
	referenceDir := t.TempDir()
	writeFiles(t, referenceDir, map[string][]byte{"go.mod": module, "main.go": []byte(reference)})
	want := fuzzRun(t, referenceDir, reference)
	inputDir := t.TempDir()
	writeFiles(t, inputDir, map[string][]byte{"go.mod": module, "main.tgo": []byte(input)})
	fuzzCompile(t, inputDir, input)
	got := fuzzRun(t, inputDir, input)
	if !bytes.Equal(got, want) {
		t.Fatalf("behavior differs\nINPUT:\n%s\nREFERENCE:\n%s\nWANT:\n%s\nGOT:\n%s\nGENERATED:\n%s", input,
			reference, want, got, readFiles(t, inputDir)["main_tgo.go"])
	}
}

func fuzzPrograms(t *testing.T, body string, data []byte, wrapped bool) (string, string) {
	t.Helper()
	values := []int{0, 1, 2, 3}
	for i := range values {
		if i+3 < len(data) {
			values[i] = int(data[i+3] % 5)
		}
	}
	failure := int(data[1])
	if failure > 3 {
		failure = -1
	}
	common := fmt.Sprintf(`package main
import ("errors"; "fmt"; "strings")
var trace strings.Builder
var failure = errors.New("failure")
var x = 0
var a = [4]int{10, 20, 30, 40}
var s = []int{10, 20, 30, 40}
var m = map[int]int{0: 10, 1: 20, 2: 30, 3: 40}
var values = [4]int{%d, %d, %d, %d}
func read(id int) int {
 fmt.Fprintf(&trace, "read%%d;", id)
 x++; a[0]++; s[0]++; m[0]++
 return values[id %% 4]
}
func readX() int { fmt.Fprint(&trace, "x;"); return x }
func at(i int) int { fmt.Fprint(&trace, "at;"); return a[i] }
func assertInt(v any) int { fmt.Fprint(&trace, "assert;"); return v.(int) }
func plus(a, b int) int { return a + b }
func sliceBase() []int { fmt.Fprint(&trace, "sliceBase;"); return s }
func mapBase() map[int]int { fmt.Fprint(&trace, "mapBase;"); return m }
func sliceIndex() int { fmt.Fprint(&trace, "slice;"); s = []int{50, 60, 70, 80}; return 0 }
func mapIndex() int { fmt.Fprint(&trace, "map;"); m = map[int]int{0: 50, 1: 60, 2: 70, 3: 80}; return 0 }
func pointer() *int { fmt.Fprint(&trace, "pointer;"); return &x }
func receiver() *item { fmt.Fprint(&trace, "receiver;"); return &item{value: x} }
func pairStep(id int) (int, int, error) { v, e := step(id); return v, v+1, e }
func step(id int) (int, error) {
 value := read(id)
 if id == %d { return value, failure }
 return value, nil
}
func emit(v ...int) { fmt.Fprintf(&trace, "emit%%v;", v) }
func truth(v int) bool { return v %% 2 != 0 }
func channel(_ int, ch chan int) chan int { return ch }
func choose(v int) func(...int) { fmt.Fprintf(&trace, "choose%%d;", v); return emit }
type item struct { value int }
func (p item) get() int { fmt.Fprint(&trace, "get;"); return p.value }
func (p *item) add(v int) { p.value += v; emit(p.value) }
func main() {
 defer func() {
  if p := recover(); p != nil { fmt.Printf("%%s|panic:%%T|x:%%d|a:%%v|s:%%v|m:%%v\n", trace.String(), p, x, a, s, m) }
 }()
 err := run()
 fmt.Printf("%%s|err:%%v|same:%%t|is:%%t|x:%%d|a:%%v|s:%%v|m:%%v\n", trace.String(), err, err == failure, errors.Is(err, failure), x, a, s, m)
}
`, values[0], values[1], values[2], values[3], failure)
	plain, err := format.Source([]byte(common + "func run() error { defer emit(8); " + body + "; return nil }"))
	if err != nil {
		t.Fatalf("invalid generator body: %v\n%s", err, body)
	}
	op := "!"
	if wrapped {
		op = "!!"
	}
	replace := func(source string, reference bool) string {
		return fuzzPropagation.ReplaceAllStringFunc(source, func(marker string) string {
			parts := fuzzPropagation.FindStringSubmatch(marker)
			call, helper := "step", "prop"
			if parts[1] == "Q" {
				call, helper = "pairStep", "propPair"
			}
			call += "(" + parts[2] + ")"
			if reference {
				return helper + "(" + call + ")"
			}
			return call + op
		})
	}
	input := replace(string(plain), false)
	refBody := replace(body, true)
	reference := common + fmt.Sprintf(`
type stop struct { err error }
func prop(value int, err error) int {
 if err != nil {
  if %t { err = fmt.Errorf("step: %%w", err) }
  panic(stop{err})
 }
 return value
}
func propPair(a, b int, err error) (int, int) {
 if err != nil {
  if %t { err = fmt.Errorf("pairStep: %%w", err) }
  panic(stop{err})
 }
 return a, b
}
func run() (err error) {
 defer func() {
  if p := recover(); p != nil {
   if stopped, ok := p.(stop); ok { err = stopped.err } else { panic(p) }
  }
 }()
 defer emit(8)
 %s
 return nil
}
`, wrapped, wrapped, refBody)
	formatted, err := format.Source([]byte(reference))
	if err != nil {
		t.Fatal(err)
	}
	return input, string(formatted)
}

func fuzzCompile(t *testing.T, dir, input string) {
	t.Helper()
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("compiler panic: %v\nINPUT:\n%s", p, input)
		}
	}()
	transpiler.Compile(dir)
}

func fuzzRun(t *testing.T, dir, input string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "-p=1", ".")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Go program failed: %v\n%s\nINPUT:\n%s", err, output, input)
	}
	return output
}

func TestFuzzProgramGeneration(t *testing.T) {
	for i, body := range fuzzBodies {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			input, reference := fuzzPrograms(t, body, []byte{byte(i), 255, 0}, false)
			if !strings.Contains(input, ")!") || strings.Contains(reference, ")!") {
				t.Fatal("operator was not replaced correctly")
			}
		})
	}
}
