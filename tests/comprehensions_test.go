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

func TestComprehensionErrors(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/comprehensions-rejected")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name(), func(t *testing.T) {
			dir := filepath.Join("testdata/comprehensions-rejected", fixture.Name())
			expected, err := os.ReadFile(filepath.Join(dir, "diagnostic.txt"))
			if err != nil {
				t.Fatal(err)
			}
			input := copyFixture(t, dir)
			var failure any
			func() { defer func() { failure = recover() }(); transpiler.Compile(input) }()
			if failure == nil || !strings.Contains(fmt.Sprint(failure), strings.TrimSpace(string(expected))) {
				t.Fatalf("got %v, want %s", failure, expected)
			}
			if !strings.Contains(fmt.Sprint(failure), "main.tgo:") {
				t.Fatalf("missing source position: %v", failure)
			}
		})
	}
}

func checkFeatureFixture(t *testing.T, dir string) {
	t.Helper()
	saved := readFiles(t, dir)
	work := t.TempDir()
	writeFiles(t, work, saved)
	transpiler.Compile(work)
	files := readFiles(t, work)
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
	for _, path := range []string{work, filepath.Join(work, "reference")} {
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
