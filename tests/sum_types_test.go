package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
