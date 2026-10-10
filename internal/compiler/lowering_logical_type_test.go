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

func TestLoweringPreservesNamedLogicalType(t *testing.T) {
	compiled, problems := Compile(PackageInput{
		Path: "logicaltype",
		Sources: []File{
			{Name: "logical_type.tgo", Data: []byte(loweringLogicalTypeSource)},
			{Name: "logical_type_test.tgo", Data: []byte(loweringLogicalTypeTestSource)},
		},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}
	directory := t.TempDir()
	for name, data := range map[string][]byte{
		"go.mod":               []byte("module logicaltype\n\ngo 1.27.0\n"),
		"logical_type.go":      compiled.Outputs["logical_type.tgo"],
		"logical_type_test.go": compiled.Outputs["logical_type_test.tgo"],
	} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-run", "TestGeneratedNamedLogicalType", "-count=1", ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

const loweringLogicalTypeSource = `package logicaltype

import "errors"

type Flag bool

var errCheck = errors.New("check failure")
var errLoad = errors.New("load failure")

func check(events *[]string, value Flag, fail bool) (Flag, error) {
	*events = append(*events, "check")
	if fail {
		return false, errCheck
	}
	return value, nil
}

func load(events *[]string, fail bool) (int, error) {
	*events = append(*events, "load")
	if fail {
		return 0, errLoad
	}
	return 7, nil
}

func consume(events *[]string, value Flag, number int) int {
	*events = append(*events, "consume")
	if value {
		return number
	}
	return 0
}

func use(events *[]string, a int, b int, checkFail bool, loadFail bool) (int, error) {
	return consume(
		events,
		a < b && check(events, true, checkFail)!!,
		load(events, loadFail)!!,
	), nil
}
`

const loweringLogicalTypeTestSource = `package logicaltype

import (
	"strings"
	"testing"
)

func TestGeneratedNamedLogicalType(t *testing.T) {
	events := []string{}
	value, err := use(&events, 2, 1, false, false)
	if value != 0 || err != nil || strings.Join(events, ",") != "load,consume" {
		t.Fatalf("short circuit value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = use(&events, 1, 2, false, false)
	if value != 7 || err != nil || strings.Join(events, ",") != "check,load,consume" {
		t.Fatalf("success value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = use(&events, 1, 2, true, false)
	if value != 0 || err != errCheck || strings.Join(events, ",") != "check" {
		t.Fatalf("check failure value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = use(&events, 1, 2, false, true)
	if value != 0 || err != errLoad || strings.Join(events, ",") != "check,load" {
		t.Fatalf("load failure value=%d error=%v events=%v", value, err, events)
	}
}
`
