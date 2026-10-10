package compiler

import (
	"go/importer"
	"go/token"
	"strings"
	"testing"
)

func TestLoweringConsumesPropagationInIncrementPlace(t *testing.T) {
	output := compileSourceOutput(t, `package sample

func load(events *[]string, values []int) ([]int, error) {
	*events = append(*events, "load")
	return values, nil
}

func index(events *[]string) int {
	*events = append(*events, "index")
	return 0
}

func update(events *[]string, values []int) error {
	load(events, values)!![index(events)]++
	return nil
}
`)
	for _, want := range []string{
		"result, err := load(events, values)",
		"operand := result",
		"operand_1 := index(events)",
		"operand[operand_1]++",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("generated increment does not contain %q\n%s", want, output)
		}
	}

	compiled, problems := Compile(PackageInput{
		Path: "sample", Sources: []File{{Name: "sample.tgo", Data: []byte(output)}},
		FileSet: token.NewFileSet(), Importer: importer.Default(),
	})
	if len(problems) != 0 {
		t.Fatalf("recheck generated increment: %v", problems[0])
	}
	_ = compiled
}
