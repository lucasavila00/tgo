package compiler

import (
	"go/importer"
	"go/token"
	"strings"
	"testing"
)

func TestEnumNamespaceLowersToGoABI(t *testing.T) {
	t.Parallel()
	data := []byte(`package sample

type Event enum {
	Ready struct {
		value *int
		labels []string = []string{"new"}
	}
	Empty struct{}
}

var ready = Event.Ready{value: nil, ..default}
var empty = Event.Empty{}
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
		"func NewEventEmpty() Event",
		"type TgoEventReadyInput struct",
		"NewEventReady(tgoInput.FieldValue, tgoInput.FieldLabels)",
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
			if len(problems) == 0 || !strings.Contains(problems[0].Error(), test.want) {
				t.Fatalf("error = %v, want %q", problems, test.want)
			}
		})
	}
}

func TestEnumGeneratedConstructorNameIsReserved(t *testing.T) {
	t.Parallel()
	data := []byte(`package sample
type Event enum { Ready struct { value string } }
func NewEventReady(value string) Event { return Event.Ready{value: value} }
`)
	_, problems := Compile(PackageInput{
		Path: "sample", Sources: []File{{Name: "sample.tgo", Data: data}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	want := "name NewEventReady is reserved by enum Event"
	if len(problems) == 0 || !strings.Contains(problems[0].Error(), want) {
		t.Fatalf("error = %v, want %q", problems, want)
	}
}
