package compiler

import (
	"context"
	"go/importer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLoweringPreservesPlacePanicOrder(t *testing.T) {
	compiled, problems := Compile(PackageInput{
		Path: "placepanic",
		Sources: []File{
			{Name: "place_panic.tgo", Data: []byte(loweringPlacePanicSource)},
			{Name: "place_panic_test.tgo", Data: []byte(loweringPlacePanicTestSource)},
		},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}
	directory := t.TempDir()
	for name, data := range map[string][]byte{
		"go.mod":              []byte("module placepanic\n\ngo 1.27.0\n"),
		"place_panic.go":      compiled.Outputs["place_panic.tgo"],
		"place_panic_test.go": compiled.Outputs["place_panic_test.tgo"],
		"native_test.go":      []byte(loweringPlacePanicNativeSource),
	} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-run", "TestGeneratedPlacePanicOrder", "-count=1", ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

const loweringPlacePanicSource = `package placepanic

import "errors"

type item struct {
	Field int
}

var errLoad = errors.New("load failure")

func record(events *[]string, event string) {
	*events = append(*events, event)
}

func mark(events *[]string) int {
	record(events, "mark")
	return 1
}

func load(events *[]string, fail bool) (int, error) {
	record(events, "load")
	if fail {
		return 0, errLoad
	}
	return 2, nil
}

func propagatedAssignment(events *[]string, state *int, fail bool) (panicked bool, err error) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	values := []item{{}}
	index := 1
	*state, values[index].Field = mark(events), load(events, fail)!!
	return false, nil
}
`

const loweringPlacePanicTestSource = `package placepanic

import (
	"strings"
	"testing"
)

var nativeAssignment func(*[]string, *int) bool

func TestGeneratedPlacePanicOrder(t *testing.T) {
	nativeEvents := []string{}
	nativeState := 0
	nativePanicked := nativeAssignment(&nativeEvents, &nativeState)

	events := []string{}
	state := 0
	panicked, err := propagatedAssignment(&events, &state, false)
	if err != nil || panicked != nativePanicked || state != nativeState ||
		strings.Join(events, ",") != strings.Join(nativeEvents, ",") {
		t.Fatalf("success events=%v state=%d panic=%v error=%v; native events=%v state=%d panic=%v",
			events, state, panicked, err, nativeEvents, nativeState, nativePanicked)
	}
	if strings.Join(events, ",") != "mark,load" || state != 1 || !panicked {
		t.Fatalf("unexpected native parity events=%v state=%d panic=%v", events, state, panicked)
	}

	events = nil
	state = 0
	panicked, err = propagatedAssignment(&events, &state, true)
	if err != errLoad || panicked || state != 0 || strings.Join(events, ",") != "mark,load" {
		t.Fatalf("failure events=%v state=%d panic=%v error=%v", events, state, panicked, err)
	}
}
`

const loweringPlacePanicNativeSource = `package placepanic

func nativeLoad(events *[]string) int {
	record(events, "load")
	return 2
}

func init() {
	nativeAssignment = runNativeAssignment
}

func runNativeAssignment(events *[]string, state *int) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	values := []item{{}}
	index := 1
	*state, values[index].Field = mark(events), nativeLoad(events)
	return false
}
`
