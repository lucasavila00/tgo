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

func TestLoweringPreservesContextualTypeIdentity(t *testing.T) {
	compiled, problems := Compile(PackageInput{
		Path: "typeidentity",
		Sources: []File{
			{Name: "type_identity.tgo", Data: []byte(loweringTypeIdentitySource)},
			{Name: "type_identity_test.tgo", Data: []byte(loweringTypeIdentityTestSource)},
		},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("compile TGo fixture: %v", problems[0])
	}
	directory := t.TempDir()
	for name, data := range map[string][]byte{
		"go.mod":                []byte("module typeidentity\n\ngo 1.27.0\n"),
		"type_identity.go":      compiled.Outputs["type_identity.tgo"],
		"type_identity_test.go": compiled.Outputs["type_identity_test.tgo"],
	} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-run", "TestGeneratedTypeIdentity", "-count=1", ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run generated Go: %v\n%s", err, output)
	}
}

const loweringTypeIdentitySource = `package typeidentity

import "errors"

type Flag bool

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

func consume(events *[]string, value Flag, number int) int {
	*events = append(*events, "consume")
	if value {
		return number
	}
	return 0
}

func use(events *[]string, a int, b int, fail bool) (int, error) {
	Flag := 1
	return consume(events,
		mark(events, "left", a) < mark(events, "right", b),
		load(events, fail)!!,
	) + Flag, nil
}

func useLocalType(events *[]string, a int, b int, fail bool) (int, error) {
	type Flag bool
	consume := func(value Flag, number int) int {
		*events = append(*events, "local-consume")
		if value {
			return number
		}
		return 0
	}
	return consume(
		mark(events, "local-left", a) < mark(events, "local-right", b),
		load(events, fail)!!,
	), nil
}

func useSiblingTypes(events *[]string) (int, error) {
	result := 0
	{
		type Flag bool
		consume := func(value Flag, number int) int {
			*events = append(*events, "first-consume")
			if value {
				return number
			}
			return 0
		}
		result += consume(
			mark(events, "first-left", 1) < mark(events, "first-right", 2),
			load(events, false)!!,
		)
	}
	{
		type Flag bool
		consume := func(value Flag, number int) int {
			*events = append(*events, "second-consume")
			if value {
				return number
			}
			return 0
		}
		result += consume(
			mark(events, "second-left", 2) < mark(events, "second-right", 1),
			load(events, false)!!,
		)
	}
	return result, nil
}
`

const loweringTypeIdentityTestSource = `package typeidentity

import (
	"strings"
	"testing"
)

func TestGeneratedTypeIdentity(t *testing.T) {
	events := []string{}
	value, err := use(&events, 1, 2, false)
	if value != 8 || err != nil || strings.Join(events, ",") != "left,right,load,consume" {
		t.Fatalf("success value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = use(&events, 1, 2, true)
	if value != 0 || err != errLoad || strings.Join(events, ",") != "left,right,load" {
		t.Fatalf("failure value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = useLocalType(&events, 1, 2, false)
	if value != 7 || err != nil || strings.Join(events, ",") != "local-left,local-right,load,local-consume" {
		t.Fatalf("local type value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = useSiblingTypes(&events)
	want := "first-left,first-right,load,first-consume,second-left,second-right,load,second-consume"
	if value != 7 || err != nil || strings.Join(events, ",") != want {
		t.Fatalf("sibling types value=%d error=%v events=%v", value, err, events)
	}
}
`
