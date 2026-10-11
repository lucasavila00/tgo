package tests

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

var updateSnapshots = flag.Bool("update-snapshots", false, "update generated Go snapshots")

func TestPropagationFixtures(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/propagation")
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no propagation fixtures")
	}
	for _, fixture := range fixtures {
		if !fixture.IsDir() {
			continue
		}
		t.Run(fixture.Name(), func(t *testing.T) {
			dir := filepath.Join("testdata/propagation", fixture.Name())
			input, err := filepath.Abs(filepath.Join(dir, "input"))
			if err != nil {
				t.Fatal(err)
			}
			files := transpiler.Compile(input)
			snapshot := filepath.Join(dir, "output")
			if *updateSnapshots {
				if err := os.RemoveAll(snapshot); err != nil {
					t.Fatal(err)
				}
				writeFiles(t, snapshot, files)
			}
			expected := readFiles(t, snapshot)
			if len(files) != len(expected) {
				t.Fatalf("generated %d files, snapshot has %d", len(files), len(expected))
			}
			for name, content := range files {
				saved, exists := expected[name]
				if !exists || !bytes.Equal(content, saved) {
					t.Errorf("generated %s differs from snapshot; inspect output before using -update-snapshots", name)
				}
			}
			output := t.TempDir()
			writeFiles(t, output, files)
			command := exec.Command("go", "run", ".")
			command.Dir = output
			actual, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("run generated source: %v\n%s", err, actual)
			}
			want, err := os.ReadFile(filepath.Join(dir, "stdout.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actual, want) {
				t.Fatalf("output: %q, want %q", actual, want)
			}
			for name := range files {
				if strings.HasSuffix(name, "_test.go") {
					command := exec.Command("go", "test", "./...")
					command.Dir = output
					actual, err := command.CombinedOutput()
					if err != nil {
						t.Fatalf("generated tests: %v\n%s", err, actual)
					}
					break
				}
			}
		})
	}
}

func writeFiles(t *testing.T, dir string, files map[string][]byte) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func readFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[name] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
