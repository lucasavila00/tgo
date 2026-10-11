package driver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileRestoresOutputsOnFailure(t *testing.T) {
	for _, input := range []string{"package p\nvar Value = missing\n", "package p\nvar Value = (\n"} {
		t.Run(input, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"go.mod":           "module example.com/rollback\n\ngo 1.27.1\n",
				"main.tgo":         input,
				"main_tgo.go":      "previous generated output\n",
				"extra.tgo":        "package p\nvar Extra = 2\n",
				"main_test.tgo":    "package p\n",
				"main_tgo_test.go": "previous test output\n",
				"inactive.tgo":     "//go:build ignore\n\npackage p\n",
				"inactive_tgo.go":  "previous excluded output\n",
			}
			for name, content := range files {
				if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0640); err != nil {
					t.Fatal(err)
				}
			}
			var failure any
			func() { defer func() { failure = recover() }(); Compile(root) }()
			if failure == nil || !strings.Contains(fmt.Sprint(failure), "main.tgo") {
				t.Fatalf("source failure: %v", failure)
			}
			for name, want := range files {
				got, err := os.ReadFile(filepath.Join(root, name))
				if err != nil || !bytes.Equal(got, []byte(want)) {
					t.Fatalf("changed %s after failure: %v, %q", name, err, got)
				}
				info, err := os.Stat(filepath.Join(root, name))
				if err != nil || info.Mode().Perm() != 0640 {
					t.Fatalf("permissions of %s: %v", name, err)
				}
			}
			if _, err := os.Stat(filepath.Join(root, "extra_tgo.go")); !os.IsNotExist(err) {
				t.Fatalf("new output remains: %v", err)
			}
		})
	}
}

func TestRestoreOutputsAfterWriteAndRemoval(t *testing.T) {
	root := t.TempDir()
	original := []byte("old output\n")
	path := filepath.Join(root, "main_tgo.go")
	if err := os.WriteFile(path, original, 0640); err != nil {
		t.Fatal(err)
	}
	source := &moduleFiles{root: root, names: []string{"main.tgo", "new.tgo"}}
	outputs := saveOutputs(source)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(root, "new_tgo.go")
	if err := os.WriteFile(newPath, []byte("new output"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := restoreOutputs(outputs); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("restored contents: %v, %q", err, got)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("restored permissions: %v", err)
	}
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatalf("new output remains: %v", err)
	}
}

func TestRestoreOutputsContinuesAfterError(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked_tgo.go")
	if err := os.Mkdir(blocked, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "main_tgo.go")
	outputs := []savedOutput{
		{path: blocked, exists: true, content: []byte("blocked"), mode: 0644},
		{path: path, exists: true, content: []byte("old output"), mode: 0644},
	}
	err := restoreOutputs(outputs)
	if err == nil || !strings.Contains(err.Error(), blocked) {
		t.Fatalf("rollback error: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "old output" {
		t.Fatalf("remaining output was not restored: %v, %q", err, got)
	}
}
