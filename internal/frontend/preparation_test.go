package frontend

import (
	"go/token"
	"reflect"
	"testing"
)

func TestSyntaxPlansKeepSourceUntilApplied(t *testing.T) {
	for name, input := range map[string]string{
		"conditional":   "package p; var x = if true { 1 } else { 2 }",
		"comprehension": "package p; var x = []int{n for _, n := range ns}",
		"short return":  "package p; func f() (int, error) { return .. err }",
	} {
		t.Run(name, func(t *testing.T) {
			source := originalSource([]byte(input))
			beforeOffsets := append([]int(nil), source.offsets...)
			original := token.NewFileSet().AddFile("main.tgo", -1, len(input))
			used := make(map[string]bool)
			var edit sourceEdit
			switch name {
			case "conditional":
				replacements := (conditionalSyntax{filename: "main.tgo", original: original, used: used}).prepare(source)
				if len(replacements) != 1 {
					t.Fatalf("conditional plans: %d", len(replacements))
				}
				edit = replacements[0].edit
			case "comprehension":
				replacement, err := source.comprehension(original, used)
				if err != nil || replacement == nil {
					t.Fatalf("comprehension plan: %v", err)
				}
				edit = replacement.edit
			case "short return":
				replacement, err := source.shortReturn(original, used)
				if err != nil || replacement == nil {
					t.Fatalf("short return plan: %v", err)
				}
				edit = replacement.edit
			}
			if source.text != input || !reflect.DeepEqual(source.offsets, beforeOffsets) {
				t.Fatal("syntax planner changed its input")
			}
			source.apply(edit)
			if source.text == input || len(source.offsets) != len(source.text)+1 {
				t.Fatal("applied plan did not produce mapped source")
			}
		})
	}
}
