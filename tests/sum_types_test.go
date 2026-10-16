package tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

// TestSumTypes compares enum behavior with independent Go programs and checks generated files.
func TestSumTypes(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/sum-types")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		if fixture.IsDir() {
			t.Run(fixture.Name(), func(t *testing.T) {
				checkFeatureFixture(t, filepath.Join("testdata/sum-types", fixture.Name()))
			})
		}
	}
}

// TestSumTypeErrors checks source positions and messages for invalid enum programs.
func TestSumTypeErrors(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/sum-types-rejected")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		if fixture.IsDir() {
			t.Run(fixture.Name(), func(t *testing.T) {
				checkSourceDiagnostic(t, filepath.Join("testdata/sum-types-rejected", fixture.Name()))
			})
		}
	}
}

// TestSumTypeDependencyFiles checks that compilation leaves the dependency module unchanged.
func TestSumTypeDependencyFiles(t *testing.T) {
	compiled := compileFixture(t, "testdata/sum-types/ExternalModule")
	for name, before := range compiled.original {
		if strings.HasPrefix(name, "dependency/") && !bytes.Equal(before, compiled.generated[name]) {
			t.Errorf("compilation changed dependency file %s", name)
		}
	}
	for name := range compiled.generated {
		if strings.HasPrefix(name, "dependency/") {
			if _, found := compiled.original[name]; !found {
				t.Errorf("compilation added dependency file %s", name)
			}
		}
	}
}

// TestSumTypeDependencySnapshot compiles the dependency from its own inputs before checking its saved Go output.
func TestSumTypeDependencySnapshot(t *testing.T) {
	directory := "testdata/sum-types/ExternalModule/dependency"
	compiled := compileFixture(t, directory)
	for name, content := range compiled.generated {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if *updateSnapshots {
			writeFiles(t, directory, map[string][]byte{name: content})
		} else if !bytes.Equal(compiled.original[name], content) {
			t.Errorf("generated dependency %s differs from snapshot:\n%s", name, content)
		}
	}
}

// TestSumTypeTagWidth checks generated tags and zero values at the uint8 limit.
func TestSumTypeTagWidth(t *testing.T) {
	for _, count := range []int{256, 257} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var source strings.Builder
			source.WriteString("package main\nimport (\"fmt\"; \"unsafe\")\ntype Choice enum {\n")
			for index := range count {
				fmt.Fprintf(&source, "V%d struct {}\n", index)
			}
			fmt.Fprintf(&source, `}
func main() {
	last := Choice.V%d{}
	var zero Choice
	fmt.Println(last.Tag(), zero.Tag(), last == zero, zero == Choice.V0{}, unsafe.Sizeof(last.Tag()))
}
`, count-1)
			root := t.TempDir()
			writeFiles(t, root, map[string][]byte{
				"go.mod":   []byte("module example.com/tagwidth\n\ngo 1.27.1\n"),
				"main.tgo": []byte(source.String()),
			})
			transpiler.Compile(root)
			command := exec.Command("go", "run", ".")
			command.Dir = root
			output, err := command.CombinedOutput()
			width := 1
			if count == 257 {
				width = 2
			}
			want := fmt.Sprintf("%d 0 false true %d\n", count-1, width)
			if err != nil || string(output) != want {
				t.Fatalf("tag output: %v, %q; want %q", err, output, want)
			}
		})
	}
}
