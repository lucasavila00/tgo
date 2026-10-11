package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestValueExpressions(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/value-expressions")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		if !fixture.IsDir() {
			continue
		}
		t.Run(fixture.Name(), func(t *testing.T) {
			dir := filepath.Join("testdata/value-expressions", fixture.Name())
			saved := readFiles(t, dir)
			work := t.TempDir()
			writeFiles(t, work, saved)
			files := transpiler.Compile(work)
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
			output := t.TempDir()
			writeFiles(t, output, files)
			want, err := os.ReadFile(filepath.Join(dir, "stdout.txt"))
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{output, filepath.Join(output, "reference")} {
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
		})
	}
}
