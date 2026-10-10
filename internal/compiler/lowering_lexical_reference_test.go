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

func TestLoweringPreservesLexicalTypeReferences(t *testing.T) {
	compiled, problems := Compile(PackageInput{
		Path: "lexicalreference",
		Sources: []File{
			{Name: "lexical_reference.tgo", Data: []byte(loweringLexicalReferenceSource)},
			{Name: "lexical_reference_test.tgo", Data: []byte(loweringLexicalReferenceTestSource)},
		},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}
	directory := t.TempDir()
	for name, data := range map[string][]byte{
		"go.mod":                    []byte("module lexicalreference\n\ngo 1.27.0\n"),
		"lexical_reference.go":      compiled.Outputs["lexical_reference.tgo"],
		"lexical_reference_test.go": compiled.Outputs["lexical_reference_test.tgo"],
	} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-run", "TestGeneratedLexicalTypeReferences", "-count=1", ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

const loweringLexicalReferenceSource = `package lexicalreference

import "errors"

var errLoad = errors.New("load failure")

func mark(events *[]string, event string, value int) int {
	*events = append(*events, event)
	return value
}

func load(events *[]string, fail bool) (int, error) {
	*events = append(*events, "load")
	if fail {
		return 0, errLoad
	}
	return 7, nil
}

func localFlag(events *[]string, a int, b int, fail bool) (int, error) {
	type Flag bool
	consume := func(value Flag, number int) int {
		*events = append(*events, "consume")
		if value {
			return number
		}
		return 0
	}
	{
		Flag := 1
		return consume(
			mark(events, "left", a) < mark(events, "right", b),
			load(events, fail)!!,
		) + Flag, nil
	}
}

func localAnonymousStruct(events *[]string, fail bool) (int, error) {
	type T int
	makeValue := func() struct{ Value T } {
		*events = append(*events, "make")
		return struct{ Value T }{Value: 3}
	}
	consume := func(value struct{ Value T }, number int) int {
		*events = append(*events, "consume")
		return int(value.Value) + number
	}
	{
		T := 1
		return consume(makeValue(), load(events, fail)!!) + T, nil
	}
}

`

const loweringLexicalReferenceTestSource = `package lexicalreference

import (
	"strings"
	"testing"
)

func TestGeneratedLexicalTypeReferences(t *testing.T) {
	events := []string{}
	value, err := localFlag(&events, 1, 2, false)
	if value != 8 || err != nil || strings.Join(events, ",") != "left,right,load,consume" {
		t.Fatalf("flag success value=%d error=%v events=%v", value, err, events)
	}
	events = nil
	value, err = localFlag(&events, 1, 2, true)
	if value != 0 || err != errLoad || strings.Join(events, ",") != "left,right,load" {
		t.Fatalf("flag failure value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = localAnonymousStruct(&events, false)
	if value != 11 || err != nil || strings.Join(events, ",") != "make,load,consume" {
		t.Fatalf("struct success value=%d error=%v events=%v", value, err, events)
	}
	events = nil
	value, err = localAnonymousStruct(&events, true)
	if value != 0 || err != errLoad || strings.Join(events, ",") != "make,load" {
		t.Fatalf("struct failure value=%d error=%v events=%v", value, err, events)
	}
}
`
