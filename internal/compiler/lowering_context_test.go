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

func TestLoweringPreservesCrossContextRuntimeSemantics(t *testing.T) {
	t.Parallel()
	compiled, problems := Compile(PackageInput{
		Path: "contexttest",
		Sources: []File{
			{Name: "context.tgo", Data: []byte(loweringContextSource)},
			{Name: "context_test.tgo", Data: []byte(loweringContextTestSource)},
		},
		FileSet:  token.NewFileSet(),
		Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}

	directory := t.TempDir()
	files := map[string][]byte{
		"go.mod":          []byte("module contexttest\n\ngo 1.27.0\n"),
		"context.go":      compiled.Outputs["context.tgo"],
		"context_test.go": compiled.Outputs["context_test.tgo"],
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-run", "TestGeneratedLoweringContexts", "-count=1", ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

const loweringContextSource = `package contexttest

import "errors"

type numbers []int
type sourceValues []int

var errFixture = errors.New("fixture failure")

func record(events *[]string, event string) {
	*events = append(*events, event)
}

func values(events *[]string, event string, fail bool) (sourceValues, error) {
	record(events, event)
	if fail {
		return nil, errFixture
	}
	return sourceValues{1, 2}, nil
}

func (values sourceValues) kept(events *[]string) sourceValues {
	record(events, "method")
	return values
}

func item(events *[]string, value int) int {
	record(events, "item")
	return value
}

func later() int {
	panic("later switch case ran")
}

func switchComprehension(events *[]string, first bool, fail bool) (string, error) {
	tag := 2
	if first {
		tag = 1
	}
	switch tag {
	case 1:
		return "first", nil
	case len(numbers{for _, value := range (values(events, "source", fail)!!).kept(events) {
		item(events, value)
	}}):
		return "comprehension", nil
	case later():
		return "later", nil
	}
	return "none", nil
}

func place(events *[]string, target *[1]int) *[1]int {
	record(events, "target")
	return target
}

func placeIndex(events *[]string) int {
	record(events, "index")
	return 0
}

func storeComprehension(events *[]string, target *[1]int) error {
	place(events, target)[placeIndex(events)] += len(numbers{for _, value := range (values(events, "source", false)!!).kept(events) {
		item(events, value)
	}})
	return nil
}

func post(events *[]string, value int, failAt int) (int, error) {
	record(events, "post")
	if value == failAt {
		return 0, errFixture
	}
	return value + 1, nil
}

func loopPost(events *[]string, failAt int) ([]func() int, []*int, error) {
	closures := []func() int{}
	addresses := []*int{}
outer:
	for value := 0; value < 3; value = post(events, value, failAt)!! {
		record(events, "body")
		closures = append(closures, func() int { return value })
		addresses = append(addresses, &value)
		continue outer
	}
	return closures, addresses, nil
}

func channel(events *[]string, event string, value chan numbers) (chan numbers, error) {
	record(events, event)
	return value, nil
}

func selectComprehension(events *[]string) (numbers, error) {
	send := make(chan numbers, 2)
	var receive chan numbers
outer:
	select {
	case channel(events, "send-channel", send)!! <- numbers{for _, value := range (values(events, "send-source", false)!!).kept(events) {
		item(events, value)
	}}:
		record(events, "send-body")
		result := numbers{for _, value := range (values(events, "body-source", false)!!).kept(events) {
			item(events, value)
		}}
		send <- result
		break outer
	case <-channel(events, "receive-channel", receive)!!:
		panic("nil receive selected")
	}
	record(events, "done")
	<-send
	return <-send, nil
}
`

const loweringContextTestSource = `package contexttest

import (
	"strings"
	"testing"
)

func TestGeneratedLoweringContexts(t *testing.T) {
	events := []string{}
	result, err := switchComprehension(&events, true, true)
	if result != "first" || err != nil || len(events) != 0 {
		t.Fatalf("skipped switch result=%q error=%v events=%v", result, err, events)
	}
	events = nil
	result, err = switchComprehension(&events, false, false)
	if result != "comprehension" || err != nil || strings.Join(events, ",") != "source,method,item,item" {
		t.Fatalf("switch result=%q error=%v events=%v", result, err, events)
	}
	events = nil
	result, err = switchComprehension(&events, false, true)
	if result != "" || err != errFixture || strings.Join(events, ",") != "source" {
		t.Fatalf("failed switch result=%q error=%v events=%v", result, err, events)
	}

	events = nil
	target := [1]int{5}
	err = storeComprehension(&events, &target)
	if err != nil || target[0] != 7 ||
		strings.Join(events, ",") != "target,index,source,method,item,item" {
		t.Fatalf("store target=%v error=%v events=%v", target, err, events)
	}

	events = nil
	closures, addresses, err := loopPost(&events, -1)
	if err != nil || strings.Join(events, ",") != "body,post,body,post,body,post" {
		t.Fatalf("loop error=%v events=%v", err, events)
	}
	for index := range 3 {
		if closures[index]() != index || *addresses[index] != index {
			t.Fatalf("capture index=%d closure=%d address=%d", index, closures[index](), *addresses[index])
		}
		for prior := range index {
			if addresses[index] == addresses[prior] {
				t.Fatalf("loop iterations %d and %d share an address", prior, index)
			}
		}
	}
	events = nil
	closures, _, err = loopPost(&events, 1)
	if err != errFixture || closures != nil || strings.Join(events, ",") != "body,post,body,post" {
		t.Fatalf("failed loop closures=%d error=%v events=%v", len(closures), err, events)
	}

	events = nil
	selected, err := selectComprehension(&events)
	if err != nil || len(selected) != 2 || selected[0] != 1 || selected[1] != 2 ||
		strings.Join(events, ",") != "send-channel,send-source,method,item,item,receive-channel,send-body,body-source,method,item,item,done" {
		t.Fatalf("select result=%v error=%v events=%v", selected, err, events)
	}
}
`
