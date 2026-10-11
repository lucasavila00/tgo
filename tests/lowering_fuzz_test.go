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
	`emit(x + P(1))`,
	`emit(P(1) + x)`,
	`emit(a[P(1)], read(2))`,
	`emit(a[read(0)], P(1))`,
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
	`emit(a[4], P(1))`,
	`var v any = "text"; emit(v.(int), P(1))`,
	`f := choose(read(0)); f(P(1), read(2))`,
	`obj := item{value: 10}; obj.add(P(1)); emit(obj.value)`,
	`obj := item{value: 10}; emit(obj.value, P(1)); emit(obj.value)`,
	`L: for i := 0; i < 2; i++ { if truth(P(1)) { continue L }; emit(i) }`,
}

var fuzzPropagation = regexp.MustCompile(`P\(([0-9]+)\)`)

func FuzzPropagationBehavior(f *testing.F) {
	for body := range fuzzBodies {
		// No error, an early error, and a later error for each context.
		for _, failure := range []byte{255, 1, 2} {
			f.Add([]byte{byte(body), failure, 0, 0, 1, 2, 3})
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 3 {
			return
		}
		body := fuzzBodies[int(data[0])%len(fuzzBodies)]
		wrapped := data[2]&1 != 0
		input, reference := fuzzPrograms(t, body, data, wrapped)
		referenceDir := t.TempDir()
		module := "module example.com/fuzz\n\ngo " + strings.TrimPrefix(runtime.Version(), "go") + "\n"
		writeFiles(t, referenceDir, map[string][]byte{"go.mod": []byte(module), "main.go": []byte(reference)})
		want := fuzzRun(t, referenceDir, reference)
		inputDir := t.TempDir()
		writeFiles(t, inputDir, map[string][]byte{"go.mod": []byte(module), "main.tgo": []byte(input)})
		files := fuzzCompile(t, inputDir, input)
		outputDir := t.TempDir()
		writeFiles(t, outputDir, files)
		got := fuzzRun(t, outputDir, input)
		if !bytes.Equal(got, want) {
			t.Fatalf("behavior differs\nINPUT:\n%s\nREFERENCE:\n%s\nWANT:\n%s\nGOT:\n%s\nGENERATED:\n%s", input, reference, want, got, files["main_tgo.go"])
		}
	})
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
	op := "!!"
	if wrapped {
		op = "!"
	}
	input := fuzzPropagation.ReplaceAllString(string(plain), "step($1)"+op)
	refBody := fuzzPropagation.ReplaceAllString(body, "prop(step($1))")
	reference := common + fmt.Sprintf(`
type stop struct { err error }
func prop(value int, err error) int {
 if err != nil {
  if %t { err = fmt.Errorf("step: %%w", err) }
  panic(stop{err})
 }
 return value
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
`, wrapped, refBody)
	formatted, err := format.Source([]byte(reference))
	if err != nil {
		t.Fatal(err)
	}
	return input, string(formatted)
}

func fuzzCompile(t *testing.T, dir, input string) (files map[string][]byte) {
	t.Helper()
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("compiler panic: %v\nINPUT:\n%s", p, input)
		}
	}()
	return transpiler.Compile(dir)
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
			if !strings.Contains(input, "!!") || strings.Contains(reference, "!!") {
				t.Fatal("operator was not replaced correctly")
			}
		})
	}
}
