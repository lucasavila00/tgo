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

func TestLoweringIncrementRuntimeOrder(t *testing.T) {
	compiled, problems := Compile(PackageInput{
		Path: "incrementruntime",
		Sources: []File{
			{Name: "increment.tgo", Data: []byte(loweringIncrementRuntimeSource)},
			{Name: "increment_test.tgo", Data: []byte(loweringIncrementRuntimeTestSource)},
		},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}
	directory := t.TempDir()
	for name, data := range map[string][]byte{
		"go.mod":            []byte("module incrementruntime\n\ngo 1.27.0\n"),
		"increment.go":      compiled.Outputs["increment.tgo"],
		"increment_test.go": compiled.Outputs["increment_test.tgo"],
	} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-run", "TestGeneratedIncrementRuntime", "-count=1", ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

const loweringIncrementRuntimeSource = `package incrementruntime

import "errors"

var errLoad = errors.New("load failure")

func load(events *[]string, values []int) ([]int, error) {
	*events = append(*events, "load")
	if values[0] < 0 {
		return nil, errLoad
	}
	return values, nil
}

func index(events *[]string) int {
	*events = append(*events, "index")
	return 0
}

func update(events *[]string, values []int) error {
	load(events, values)!![index(events)]++
	return nil
}
`

const loweringIncrementRuntimeTestSource = `package incrementruntime

import (
	"strings"
	"testing"
)

func TestGeneratedIncrementRuntime(t *testing.T) {
	events := []string{}
	values := []int{4}
	err := update(&events, values)
	if err != nil || values[0] != 5 || strings.Join(events, ",") != "load,index" {
		t.Fatalf("success values=%v error=%v events=%v", values, err, events)
	}

	events = nil
	values = []int{-1}
	err = update(&events, values)
	if err != errLoad || values[0] != -1 || strings.Join(events, ",") != "load" {
		t.Fatalf("failure values=%v error=%v events=%v", values, err, events)
	}
}
`
