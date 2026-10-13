package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCollectionComprehensions checks comprehension snapshots and runtime results against Go reference programs.
func TestCollectionComprehensions(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/comprehensions")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		if !fixture.IsDir() {
			continue
		}
		t.Run(fixture.Name(), func(t *testing.T) {
			checkFeatureFixture(t, filepath.Join("testdata/comprehensions", fixture.Name()))
		})
	}
}

// TestComprehensionErrors checks source diagnostics for invalid comprehensions.
func TestComprehensionErrors(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/comprehensions-rejected")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name(), func(t *testing.T) {
			checkSourceDiagnostic(t, filepath.Join("testdata/comprehensions-rejected", fixture.Name()))
		})
	}
}

// checkFeatureFixture checks generated snapshots and compares compiled and reference program output.
func checkFeatureFixture(t *testing.T, dir string) {
	t.Helper()
	compiled := compileFixture(t, dir)
	saved, files := compiled.original, compiled.generated
	for name, content := range files {
		if !strings.HasSuffix(name, ".go") || strings.HasPrefix(name, "reference/") {
			continue
		}
		if *updateSnapshots {
			writeFiles(t, dir, map[string][]byte{name: content})
		} else if !bytes.Equal(saved[name], content) {
			t.Errorf("generated %s differs from snapshot:\n%s", name, content)
		}
	}
	want, err := os.ReadFile(filepath.Join(dir, "stdout.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{compiled.directory, filepath.Join(compiled.directory, "reference")} {
		command := exec.Command("go", "run", ".")
		command.Dir = path
		actual, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("run %s: %v\n%s", path, err, actual)
		}
		if !bytes.Equal(actual, want) {
			t.Fatalf("run %s: %q, want %q", path, actual, want)
		}
	}
}
