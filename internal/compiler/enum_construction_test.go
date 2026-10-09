package compiler

import (
	"fmt"
	"go/importer"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestEnumNamespaceLowersToGoABI(t *testing.T) {
	t.Parallel()
	data := []byte(`package sample

import "net/http"

type Event enum {
	Ready struct {
		value *int
		labels []string = []string{"new"}
	}
	Named struct { Event string; EventNamed string; EventTagNamed string }
	Empty struct{}
}

type Request enum { Value struct { *http.Request } }

var ready = Event.Ready{value: nil, ..default}
var empty = Event.Empty{}

func shadowPayloadName() Event {
	type EventReady struct{}
	return Event.Ready{value: nil, ..default}
}
`)
	compiled, problems := Compile(PackageInput{
		Path: "sample", Sources: []File{{Name: "sample.tgo", Data: data}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	for _, text := range []string{
		"func NewEventReady(value *int, labels []string) Event",
		"func NewEventNamed(tgoField0 string, tgoField1 string, tgoField2 string) Event",
		"func NewRequestValue(tgoField0 *http.Request) Request",
		"func NewEventEmpty() Event",
		"type TgoEventReadyInput struct",
		"NewEventReady(input.FieldValue, input.FieldLabels)",
		"NewEventEmpty()",
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("generated enum does not contain %q\n%s", text, output)
		}
	}
	if strings.Contains(output, "func (value EventReady) Event() Event") {
		t.Fatal("generated enum contains a payload conversion method")
	}
}

func TestEnumGeneratedConstructionSurfaceIsPrivateToTGo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		use  string
		want string
	}{
		{
			name: "constructor",
			use:  `var _ = NewEventReady("value")`,
			want: "NewEventReady is generated Go ABI; use Event.Ready{...}",
		},
		{
			name: "constructor function value",
			use: `func makeEvent() Event {
	constructor := NewEventReady
	return constructor("value")
}`,
			want: "NewEventReady is generated Go ABI; use Event.Ready{...}",
		},
		{
			name: "payload",
			use:  `var _ EventReady`,
			want: "EventReady is generated enum representation; use Event.Ready{...}",
		},
		{
			name: "carrier",
			use:  `var _ TgoEventReadyInput`,
			want: "TgoEventReadyInput is generated staging ABI; use Event.Ready{...}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			data := []byte("package sample\n" +
				"type Event enum { Ready struct { value string } }\n" + test.use + "\n")
			_, problems := Compile(PackageInput{
				Path: "sample", Sources: []File{{Name: "sample.tgo", Data: data}},
				FileSet: token.NewFileSet(), Importer: importer.Default(),
			})
			if len(problems) != 1 || !strings.Contains(problems[0].Error(), test.want) {
				t.Fatalf("error = %v, want %q", problems, test.want)
			}
		})
	}
}

func TestImportedEnumConstructionSurface(t *testing.T) {
	t.Parallel()
	dependency, problems := Compile(PackageInput{
		Path: "example.test/dep",
		Sources: []File{{Name: "dep.tgo", Data: []byte(`package dep
type Event enum {
	Ready struct { value string; pointer *int }
	Empty struct{}
}
`)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	load := enumTestImporter{
		path: "example.test/dep", pkg: dependency.Package,
		fallback: importer.Default(),
	}
	valid := []byte(`package sample
import "example.test/dep"
var ready = dep.Event.Ready{value: "value", pointer: nil}
var empty = dep.Event.Empty{}
`)
	compiled, problems := Compile(PackageInput{
		Path: "sample", Sources: []File{{Name: "sample.tgo", Data: valid}},
		Imports: map[string]*CompiledPackage{"example.test/dep": dependency},
		FileSet: token.NewFileSet(), Importer: load,
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	for _, text := range []string{"dep.NewEventReady", "dep.NewEventEmpty"} {
		if !strings.Contains(output, text) {
			t.Fatalf("imported enum output does not contain %q\n%s", text, output)
		}
	}

	tests := []struct {
		name string
		use  string
		want string
	}{
		{
			name: "constructor",
			use:  `var _ = dep.NewEventReady("value", nil)`,
			want: "NewEventReady is generated Go ABI; use Event.Ready{...}",
		},
		{
			name: "constructor function value",
			use: `func makeEvent() dep.Event {
	constructor := dep.NewEventReady
	return constructor("value", nil)
}`,
			want: "NewEventReady is generated Go ABI; use Event.Ready{...}",
		},
		{
			name: "payload",
			use:  `var _ dep.EventReady`,
			want: "EventReady is generated enum representation; use Event.Ready{...}",
		},
		{
			name: "carrier",
			use:  `var _ dep.TgoEventReadyInput`,
			want: "TgoEventReadyInput is generated staging ABI; use Event.Ready{...}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			data := []byte("package sample\nimport \"example.test/dep\"\n" + test.use + "\n")
			_, problems := Compile(PackageInput{
				Path: "sample", Sources: []File{{Name: "sample.tgo", Data: data}},
				Imports: map[string]*CompiledPackage{"example.test/dep": dependency},
				FileSet: token.NewFileSet(), Importer: load,
			})
			if len(problems) != 1 || !strings.Contains(problems[0].Error(), test.want) {
				t.Fatalf("error = %v, want %q", problems, test.want)
			}
		})
	}
}

func TestExactEnumConstraintGetsGeneratedMethodSet(t *testing.T) {
	t.Parallel()
	data := []byte(`package sample
type Event enum {
	Ready struct { value string }
	Empty struct{}
}
type Events interface { Event }
func label[T Events](value T) string {
	switch value.Tag() {
	case EventTagReady:
		return value.ReadyPayload().value
	case EventTagEmpty:
		return ""
	exhaustive:
	}
}
`)
	compiled, problems := Compile(PackageInput{
		Path: "sample", Sources: []File{{Name: "sample.tgo", Data: data}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatal(problems[0])
	}
	output := string(compiled.Outputs["sample.tgo"])
	for _, text := range []string{
		"Tag() EventTag",
		"UnknownTag() string",
		"ReadyPayload() EventReady",
		"EmptyPayload() EventEmpty",
	} {
		if !strings.Contains(output, text) {
			t.Fatalf("exact enum constraint does not contain %q\n%s", text, output)
		}
	}
}

type enumTestImporter struct {
	path     string
	pkg      *types.Package
	fallback types.Importer
}

func (i enumTestImporter) Import(path string) (*types.Package, error) {
	if path == i.path {
		return i.pkg, nil
	}
	if i.fallback == nil {
		return nil, fmt.Errorf("cannot import %s", path)
	}
	return i.fallback.Import(path)
}

func TestEnumGeneratedNamesAreReservedAcrossFiles(t *testing.T) {
	t.Parallel()
	enumSource := File{
		Name: "enum.tgo",
		Data: []byte("package sample\ntype Event enum { Ready struct { value string } }\n"),
	}
	tests := []struct {
		name   string
		source *File
		goFile *File
		want   string
	}{
		{
			name: "TGo constructor", source: &File{
				Name: "other.tgo", Data: []byte("package sample\nfunc NewEventReady() {}\n"),
			},
			want: "other.tgo:2:6: name NewEventReady is reserved by enum Event",
		},
		{
			name: "TGo carrier", source: &File{
				Name: "other.tgo",
				Data: []byte("package sample\ntype TgoEventReadyInput struct{}\n"),
			},
			want: "other.tgo:2:6: name TgoEventReadyInput is reserved by enum Event",
		},
		{
			name: "Go constructor", goFile: &File{
				Name: "other.go", Data: []byte("package sample\nfunc NewEventReady() {}\n"),
			},
			want: "other.go:2:6: name NewEventReady is reserved by enum Event",
		},
		{
			name: "Go carrier", goFile: &File{
				Name: "other.go",
				Data: []byte("package sample\ntype TgoEventReadyInput struct{}\n"),
			},
			want: "other.go:2:6: name TgoEventReadyInput is reserved by enum Event",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			for range 3 {
				input := PackageInput{
					Path: "sample", Sources: []File{enumSource},
					FileSet: token.NewFileSet(), Importer: importer.Default(),
				}
				if test.source != nil {
					input.Sources = append(input.Sources, *test.source)
				}
				if test.goFile != nil {
					input.GoFiles = append(input.GoFiles, *test.goFile)
				}
				_, problems := Compile(input)
				if len(problems) != 1 || problems[0].Error() != test.want {
					t.Fatalf("error = %v, want %q", problems, test.want)
				}
			}
		})
	}
}
