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
			checkPropagationFixture(t, fixture.Name())
		})
	}
}

func checkPropagationSnapshots(t *testing.T, input string, source, files map[string][]byte) {
	t.Helper()
	for name, content := range files {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if *updateSnapshots {
			if !bytes.Equal(source[name], content) {
				writeFiles(t, input, map[string][]byte{name: content})
			}
			continue
		}
		if saved, exists := source[name]; !exists || !bytes.Equal(content, saved) {
			t.Errorf("generated %s differs from snapshot; inspect output before using -update-snapshots\nSAVED:\n%s\nGENERATED:\n%s", name, saved, content)
		}
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

func TestPropagationSnapshotDeterminism(t *testing.T) {
	for _, name := range []string{"Contexts--short_circuit--direct", "Contexts--short_circuit--wrapped"} {
		t.Run(name, func(t *testing.T) {
			dir := copyFixture(t, filepath.Join("testdata/propagation", name))
			transpiler.Compile(dir)
			first := readFiles(t, dir)
			for i := 0; i < 3; i++ {
				transpiler.Compile(dir)
				next := readFiles(t, dir)
				if len(next) != len(first) {
					t.Fatal("generated file set changed")
				}
				for name, content := range first {
					if !bytes.Equal(next[name], content) {
						t.Fatalf("compilation %d changed %s", i+2, name)
					}
				}
			}
		})
	}
}

func copyFixture(t *testing.T, dir string) string {
	t.Helper()
	work := t.TempDir()
	writeFiles(t, work, readFiles(t, dir))
	return work
}

func checkPropagationFixture(t *testing.T, name string) {
	t.Helper()
	dir := filepath.Join("testdata/propagation", name)
	input, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	source := readFiles(t, input)
	work := t.TempDir()
	writeFiles(t, work, source)
	transpiler.Compile(work)
	files := readFiles(t, work)
	inputs := 0
	for name := range source {
		if strings.HasSuffix(name, ".tgo") {
			inputs++
		}
	}
	if inputs == 0 {
		t.Fatal("fixture has no TGo input")
	}
	checkPropagationSnapshots(t, input, source, files)
	command := exec.Command("go", "run", ".")
	command.Dir = work
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
			command.Dir = work
			actual, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("generated tests: %v\n%s", err, actual)
			}
			break
		}
	}
}
