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

func TestLoweringPreservesEvaluationContracts(t *testing.T) {
	compiled, problems := Compile(PackageInput{
		Path: "evaluationcontract",
		Sources: []File{
			{Name: "evaluation_contract.tgo", Data: []byte(loweringEvaluationContractSource)},
			{Name: "evaluation_contract_test.tgo", Data: []byte(loweringEvaluationContractTestSource)},
		},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}
	directory := t.TempDir()
	for name, data := range map[string][]byte{
		"go.mod":                      []byte("module evaluationcontract\n\ngo 1.27.0\n"),
		"evaluation_contract.go":      compiled.Outputs["evaluation_contract.tgo"],
		"evaluation_contract_test.go": compiled.Outputs["evaluation_contract_test.tgo"],
	} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-run", "TestGeneratedEvaluationContracts", "-count=1", ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

const loweringEvaluationContractSource = `package evaluationcontract

import "errors"

var errLoad = errors.New("load failure")

func record(events *[]string, event string) {
	*events = append(*events, event)
}

type Cell struct {
	Value int
}

type Mid struct {
	*Cell
}

type Outer struct {
	*Mid
}

func factory(events *[]string, value *Outer) *Outer {
	record(events, "factory")
	return value
}

func load(events *[]string, fail bool) (int, error) {
	record(events, "load")
	if fail {
		return 0, errLoad
	}
	return 9, nil
}

func storePromoted(events *[]string, value *Outer, fail bool) error {
	factory(events, value).Value = load(events, fail)!!
	return nil
}

type BasicFlag bool

func mark(events *[]string, event string, value int) int {
	record(events, event)
	return value
}

func loadFlag(events *[]string, fail bool) (BasicFlag, error) {
	record(events, "load-flag")
	if fail {
		return false, errLoad
	}
	return true, nil
}

func declaredFlags(events *[]string, x int, y int, fail bool) (BasicFlag, BasicFlag, error) {
	var a, b BasicFlag = mark(events, "left", x) < mark(events, "right", y), loadFlag(events, fail)!!
	return a, b, nil
}

type C interface {
	~int
}

type Flag[P C] bool

func Consume[V C](events *[]string, value Flag[V], number int) int {
	record(events, "consume")
	if value {
		return number
	}
	return 0
}

func loadNumber(events *[]string, fail bool) (int, error) {
	record(events, "load-number")
	if fail {
		return 0, errLoad
	}
	return 7, nil
}

func genericUse(events *[]string, x int, y int, fail bool) (int, error) {
	type C string
	type Flag bool
	type V int
	_, _ = C("shadow"), Flag(false)
	return Consume[V](
		events,
		mark(events, "generic-left", x) < mark(events, "generic-right", y),
		loadNumber(events, fail)!!,
	), nil
}
`

const loweringEvaluationContractTestSource = `package evaluationcontract

import (
	"strings"
	"testing"
)

func TestGeneratedEvaluationContracts(t *testing.T) {
	events := []string{}
	cell := &Cell{Value: 1}
	outer := &Outer{Mid: &Mid{Cell: cell}}
	err := storePromoted(&events, outer, false)
	if err != nil || cell.Value != 9 || strings.Join(events, ",") != "factory,load" {
		t.Fatalf("promoted success value=%d error=%v events=%v", cell.Value, err, events)
	}
	events = nil
	cell.Value = 1
	err = storePromoted(&events, outer, true)
	if err != errLoad || cell.Value != 1 || strings.Join(events, ",") != "factory,load" {
		t.Fatalf("promoted failure value=%d error=%v events=%v", cell.Value, err, events)
	}

	events = nil
	a, b, err := declaredFlags(&events, 1, 2, false)
	if !a || !b || err != nil || strings.Join(events, ",") != "left,right,load-flag" {
		t.Fatalf("flags success a=%v b=%v error=%v events=%v", a, b, err, events)
	}
	events = nil
	a, b, err = declaredFlags(&events, 1, 2, true)
	if a || b || err != errLoad || strings.Join(events, ",") != "left,right,load-flag" {
		t.Fatalf("flags failure a=%v b=%v error=%v events=%v", a, b, err, events)
	}

	events = nil
	result, err := genericUse(&events, 1, 2, false)
	if result != 7 || err != nil ||
		strings.Join(events, ",") != "generic-left,generic-right,load-number,consume" {
		t.Fatalf("generic success result=%d error=%v events=%v", result, err, events)
	}
	events = nil
	result, err = genericUse(&events, 1, 2, true)
	if result != 0 || err != errLoad ||
		strings.Join(events, ",") != "generic-left,generic-right,load-number" {
		t.Fatalf("generic failure result=%d error=%v events=%v", result, err, events)
	}
}
`
