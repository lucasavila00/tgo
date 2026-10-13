package frontend

import (
	"go/token"
	"testing"
)

func TestConditionalPlanKeepsSourceUntilApplied(t *testing.T) {
	input := "package p; var x = if true { 1 } else { 2 }"
	source := originalSource([]byte(input))
	original := token.NewFileSet().AddFile("main.tgo", -1, len(input))
	syntax := conditionalSyntax{filename: "main.tgo", original: original, used: make(map[string]bool)}
	replacements := syntax.prepare(source)
	if len(replacements) != 1 {
		t.Fatalf("conditional plans: %d", len(replacements))
	}
	checkPlannedEdit(t, source, replacements[0].edit, input)
}

func TestComprehensionPlanKeepsSourceUntilApplied(t *testing.T) {
	input := "package p; var x = []int{n for _, n := range ns}"
	source := originalSource([]byte(input))
	original := token.NewFileSet().AddFile("main.tgo", -1, len(input))
	replacement, err := source.comprehension(original, make(map[string]bool))
	if err != nil || replacement == nil {
		t.Fatalf("comprehension plan: %v", err)
	}
	checkPlannedEdit(t, source, replacement.edit, input)
}

func TestShortReturnPlanKeepsSourceUntilApplied(t *testing.T) {
	input := "package p; func f() (int, error) { return .. err }"
	source := originalSource([]byte(input))
	original := token.NewFileSet().AddFile("main.tgo", -1, len(input))
	replacement, err := source.shortReturn(original, make(map[string]bool))
	if err != nil || replacement == nil {
		t.Fatalf("short return plan: %v", err)
	}
	checkPlannedEdit(t, source, replacement.edit, input)
}

func checkPlannedEdit(t *testing.T, source mappedSource, edit sourceEdit, input string) {
	t.Helper()
	if source.text != input {
		t.Fatal("syntax planner changed its input")
	}
	for offset, position := range source.offsets {
		if offset != position {
			t.Fatal("syntax planner changed input positions")
		}
	}
	source.apply(edit)
	if source.text == input || len(source.offsets) != len(source.text)+1 {
		t.Fatal("applied plan did not produce mapped source")
	}
}
