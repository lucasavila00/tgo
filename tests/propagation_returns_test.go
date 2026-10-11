package tests

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestPropagationReturnTypes(t *testing.T) {
	module, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, next, run, call, operator, from, to string
	}{
		{"interface rejected", "func next() (int, error) { return 1, richError{} }", "func run() Rich", "next()", "!", "error", "Rich"},
		{"wrapped interface rejected", "func next() (int, Rich) { return 1, richError{} }", "func run() Rich", "next()", "!!", "error", "Rich"},
		{"concrete rejected", "func next() (int, plainError) { return 1, plainError{} }", "func run() Rich", "next()", "!", "plainError", "Rich"},
		{"generic rejected", "func next[E error](err E) (int, E) { return 1, err }", "func run[E error](err E) Rich", "next(err)", "!", "E", "Rich"},
		{"generic wrapped rejected", "func next[E error](err E) (int, E) { return 1, err }", "func run[E error](err E) E", "next(err)", "!!", "error", "E"},
		{"literal rejected", "func next() (int, error) { return 1, richError{} }", "var run = func() Rich", "next()", "!", "error", "Rich"},
		{"wrapped literal rejected", "func next() (int, Rich) { return 1, richError{} }", "var run = func() Rich", "next()", "!!", "error", "Rich"},
		{"richer accepted", "func next() (int, Rich) { return 1, richError{} }", "func run() Rich", "next()", "!", "", ""},
		{"concrete accepted", "func next() (int, richError) { return 1, richError{} }", "func run() Rich", "next()", "!", "", ""},
		{"generic accepted", "func next[E error](err E) (int, E) { return 1, err }", "func run[E Rich](err E) Rich", "next(err)", "!", "", ""},
		{"generic result accepted", "func next[E error](err E) (int, E) { return 1, err }", "func run[E error](err E) E", "next(err)", "!", "", ""},
		{"generic wrapped accepted", "func next[E error](err E) (int, E) { return 1, err }", "func run[E error](err E) error", "next(err)", "!!", "", ""},
		{"literal accepted", "func next() (int, Rich) { return 1, richError{} }", "var run = func() error", "next()", "!", "", ""},
		{"wrapped accepted", "func next() (int, Rich) { return 1, richError{} }", "func run() error", "next()", "!!", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			main := "func main() {}"
			if test.from == "" {
				call := "run()"
				if strings.Contains(test.run, "[E ") {
					call = "run(richError{})"
				}
				main = "func main() { fmt.Println(errors.Is(" + call + ", richError{})) }"
			}
			returned := "return nil"
			if strings.HasSuffix(test.run, ") E") {
				returned = "var zero E; return zero"
			}
			input := `package main
import ("errors"; "fmt")
type Rich interface { error; Extra() }
type richError struct{}
func (richError) Error() string { return "failed" }
func (richError) Extra() {}
type plainError struct{}
func (plainError) Error() string { return "failed" }
var _ = errors.Is
var _ = fmt.Println
` + test.next + "\n" + test.run + " {\n    _ = " + test.call + test.operator + "\n    " + returned + "\n}\n" + main + "\n"
			dir := t.TempDir()
			writeFiles(t, dir, map[string][]byte{"go.mod": module, "main.tgo": []byte(input)})
			if test.from != "" {
				defer func() {
					actual := recover()
					if actual == nil {
						t.Fatal("expected incompatible error rejection")
					}
					position := fmt.Sprintf("main.tgo:13:%d", 9+len(test.call))
					message := fmt.Sprint(actual)
					for _, part := range []string{position, test.operator + " cannot return error of type", test.from, test.to} {
						if !strings.Contains(message, part) {
							t.Fatalf("diagnostic %q does not contain %q", message, part)
						}
					}
				}()
				transpiler.Compile(dir)
				return
			}
			transpiler.Compile(dir)
			command := exec.Command("go", "run", "-p=1", ".")
			command.Dir = dir
			actual, err := command.CombinedOutput()
			if err != nil || string(actual) != "true\n" {
				t.Fatalf("generated program: %v\n%s", err, actual)
			}
		})
	}
}
