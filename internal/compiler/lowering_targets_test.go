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

func TestLoweringTargetsPreserveRuntimeControlAndScope(t *testing.T) {
	compiled, problems := Compile(PackageInput{
		Path: "targets",
		Sources: []File{
			{Name: "targets.tgo", Data: []byte(loweringTargetsSource)},
			{Name: "targets_test.tgo", Data: []byte(loweringTargetsTestSource)},
		},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}
	directory := t.TempDir()
	for name, data := range map[string][]byte{
		"go.mod":          []byte("module targets\n\ngo 1.27.0\n"),
		"targets.go":      compiled.Outputs["targets.tgo"],
		"targets_test.go": compiled.Outputs["targets_test.tgo"],
	} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-run", "TestGeneratedLoweringTargets", "-count=1", ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

const loweringTargetsSource = `package targets

func record(events *[]string, event string) {
	*events = append(*events, event)
}

func load(events *[]string, event string, value int) (int, error) {
	record(events, event)
	return value, nil
}

func post(events *[]string, value int) int {
	record(events, "post")
	return value + 1
}

func dualRoleLoop(events *[]string) ([]func() int, []*int, error) {
	closures := []func() int{}
	addresses := []*int{}
	retry := true
loop:
	for value := load(events, "init", 0)!!; value < 2; value = post(events, value) {
		record(events, "body")
		closures = append(closures, func() int { return value })
		addresses = append(addresses, &value)
		if value == 0 {
			continue loop
		}
		break loop
	}
	if retry {
		retry = false
		record(events, "goto")
		goto loop
	}
	return closures, addresses, nil
}

func jumpsAroundLowering(events *[]string) error {
	goto forward
back:
	record(events, "back")
	return nil
forward:
	_ = load(events, "load", 1)!!
	goto back
}

func labeledDeclaration(events *[]string) (int, error) {
	runs := 0
	goto entry
entry:
	value := load(events, "load", 7)!!
	runs++
	record(events, "use")
	if runs < 2 {
		goto entry
	}
	return value, nil
}
`

const loweringTargetsTestSource = `package targets

import (
	"strings"
	"testing"
)

func TestGeneratedLoweringTargets(t *testing.T) {
	events := []string{}
	closures, addresses, err := dualRoleLoop(&events)
	wantEvents := "init,body,post,body,goto,init,body,post,body"
	if err != nil || len(closures) != 4 || len(addresses) != 4 || strings.Join(events, ",") != wantEvents {
		t.Fatalf("loop closures=%d addresses=%d error=%v events=%v", len(closures), len(addresses), err, events)
	}
	wantValues := []int{0, 1, 0, 1}
	for index, want := range wantValues {
		if closures[index]() != want || *addresses[index] != want {
			t.Fatalf("capture index=%d closure=%d address=%d want=%d", index, closures[index](), *addresses[index], want)
		}
		for prior := range index {
			if addresses[index] == addresses[prior] {
				t.Fatalf("executions %d and %d share an address", prior, index)
			}
		}
	}

	events = nil
	if err := jumpsAroundLowering(&events); err != nil || strings.Join(events, ",") != "load,back" {
		t.Fatalf("jumps error=%v events=%v", err, events)
	}

	events = nil
	value, err := labeledDeclaration(&events)
	if value != 7 || err != nil || strings.Join(events, ",") != "load,use,load,use" {
		t.Fatalf("declaration value=%d error=%v events=%v", value, err, events)
	}
}
`
