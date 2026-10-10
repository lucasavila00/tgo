package compiler

import "testing"

func TestLoweringTargetsPreserveRuntimeControlAndScope(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "targets", sourceName: "targets.tgo", source: loweringTargetsSource,
		testName: "targets_test.tgo", testSource: loweringTargetsTestSource,
		testPattern: "TestGeneratedLoweringTargets",
	})
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

type Flag bool

func loadFlag(events *[]string) (Flag, error) {
	record(events, "flag")
	return true, nil
}

func labeledNamedDeclaration(events *[]string) (Flag, error) {
	goto entry
entry:
	value := loadFlag(events)!!
	return value, nil
}

func labeledLocalNamedDeclaration(events *[]string) (bool, error) {
	type LocalFlag bool
	load := func() (LocalFlag, error) {
		record(events, "local-flag")
		return true, nil
	}
	goto entry
entry:
	value := load()!!
	consume := func(value LocalFlag) bool { return bool(value) }
	return consume(value), nil
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
	if err != nil || len(closures) != 4 || len(addresses) != 4 ||
		strings.Join(events, ",") != wantEvents {
		t.Fatalf("loop closures=%d addresses=%d error=%v events=%v",
			len(closures), len(addresses), err, events)
	}
	wantValues := []int{0, 1, 0, 1}
	for index, want := range wantValues {
		if closures[index]() != want || *addresses[index] != want {
			t.Fatalf("capture index=%d closure=%d address=%d want=%d",
				index, closures[index](), *addresses[index], want)
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

	events = nil
	flag, err := labeledNamedDeclaration(&events)
	if !flag || err != nil || strings.Join(events, ",") != "flag" {
		t.Fatalf("named declaration value=%v error=%v events=%v", flag, err, events)
	}

	events = nil
	localFlag, err := labeledLocalNamedDeclaration(&events)
	if !localFlag || err != nil || strings.Join(events, ",") != "local-flag" {
		t.Fatalf("local named declaration value=%v error=%v events=%v", localFlag, err, events)
	}
}
`
