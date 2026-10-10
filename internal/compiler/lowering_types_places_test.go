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

func TestLoweringPreservesContextualTypesAndPlaces(t *testing.T) {
	compiled, problems := Compile(PackageInput{
		Path: "typesplaces",
		Sources: []File{
			{Name: "types_places.tgo", Data: []byte(loweringTypesPlacesSource)},
			{Name: "types_places_test.tgo", Data: []byte(loweringTypesPlacesTestSource)},
		},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}
	directory := t.TempDir()
	for name, data := range map[string][]byte{
		"go.mod":               []byte("module typesplaces\n\ngo 1.27.0\n"),
		"types_places.go":      compiled.Outputs["types_places.tgo"],
		"types_places_test.go": compiled.Outputs["types_places_test.tgo"],
	} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-run", "TestGeneratedTypesAndPlaces", "-count=1", ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

const loweringTypesPlacesSource = `package typesplaces

func record(events *[]string, event string) {
	*events = append(*events, event)
}

func shift(events *[]string) (uint, error) {
	record(events, "shift")
	return 63, nil
}

func contextualSend(events *[]string) (uint64, error) {
	values := make(chan uint64, 1)
	select {
	case values <- 1 << shift(events)!!:
	}
	return <-values, nil
}

func mutate(events *[]string, value *uint) (chan int, error) {
	record(events, "mutate")
	*value = 63
	return nil, nil
}

func snapshottedSend(events *[]string) (int, uint, error) {
	values := make(chan int, 1)
	var n uint = 1
	select {
	case values <- 1 << n:
	case <-mutate(events, &n)!!:
	}
	return <-values, n, nil
}

type cell struct {
	Value int
}

type holder struct {
	Cells [1]cell
}

func selectHolder(events *[]string, value *holder) *holder {
	record(events, "target")
	return value
}

func selectIndex(events *[]string) int {
	record(events, "index")
	return 0
}

func loadValue(events *[]string) (int, error) {
	record(events, "rhs")
	return 9, nil
}

func storeDirectPlace(events *[]string, value *cell) error {
	value.Value = loadValue(events)!!
	return nil
}

func storeNestedPlace(events *[]string, value *holder) error {
	selectHolder(events, value).Cells[selectIndex(events)].Value = loadValue(events)!!
	return nil
}
`

const loweringTypesPlacesTestSource = `package typesplaces

import (
	"strings"
	"testing"
)

func TestGeneratedTypesAndPlaces(t *testing.T) {
	events := []string{}
	value, err := contextualSend(&events)
	if value != uint64(1)<<63 || err != nil || strings.Join(events, ",") != "shift" {
		t.Fatalf("contextual value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	snapshot, n, err := snapshottedSend(&events)
	if snapshot != 2 || n != 63 || err != nil || strings.Join(events, ",") != "mutate" {
		t.Fatalf("snapshot value=%d n=%d error=%v events=%v", snapshot, n, err, events)
	}

	events = nil
	direct := cell{}
	err = storeDirectPlace(&events, &direct)
	if direct.Value != 9 || err != nil || strings.Join(events, ",") != "rhs" {
		t.Fatalf("direct place=%v error=%v events=%v", direct, err, events)
	}

	events = nil
	target := holder{}
	err = storeNestedPlace(&events, &target)
	if target.Cells[0].Value != 9 || err != nil || strings.Join(events, ",") != "target,index,rhs" {
		t.Fatalf("place target=%v error=%v events=%v", target, err, events)
	}
}
`
