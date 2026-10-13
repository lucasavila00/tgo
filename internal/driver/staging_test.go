package driver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestReadModuleReadsOnlyOwnedSource checks that discovery reads only source owned by the module.
func TestReadModuleReadsOnlyOwnedSource(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":              "module example.com/source\n",
		"main.tgo":            "package p\nvar Value = 1\n",
		"ordinary.go":         "package p\nvar Keep = 2\n",
		"main_tgo.go":         "stale snapshot",
		"asset.bin":           "asset contents",
		"testdata/example.go": "fixture snapshot",
		"vendor/example.go":   "dependency source",
		"nested/go.mod":       "module example.com/nested\n",
		"nested/example.go":   "nested source",
		"nested/example.tgo":  "nested TGo source",
		".data/example.go":    "hidden source",
		"_data/example.go":    "data source",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	source := readModule(root)
	want := []sourceFile{{name: "main.tgo", content: []byte(files["main.tgo"])},
		{name: "ordinary.go", content: []byte(files["ordinary.go"])}}
	if !reflect.DeepEqual(source.files, want) {
		t.Fatalf("source files: got %v, want %v", source.files, want)
	}
}

// TestCompileWritesOnlyOwnedGeneratedFiles checks output replacement and preservation of source and assets.
func TestCompileWritesOnlyOwnedGeneratedFiles(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"go.mod":      []byte("module example.com/stages\n\ngo 1.27.1\n"),
		"main.tgo":    []byte("package p\nvar Value = if true {Keep} else {0}\n"),
		"ordinary.go": []byte("package p; var Keep=2\n"),
		"main_tgo.go": []byte("invalid snapshot"),
		"asset.bin":   bytes.Repeat([]byte{42}, 1<<20),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	Compile(root)
	for _, name := range []string{"main.tgo", "ordinary.go", "asset.bin"} {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, files[name]) {
			t.Fatalf("changed input %s: %v", name, err)
		}
	}
	got, err := os.ReadFile(filepath.Join(root, "main_tgo.go"))
	if err != nil || len(got) == 0 || bytes.Equal(got, files["main.tgo"]) {
		t.Fatalf("generated disk output: %v, %s", err, got)
	}
}

// TestCompileRemovesFailedStageAndAllowsRetry checks output removal on failure and compilation after correction.
func TestCompileRemovesFailedStageAndAllowsRetry(t *testing.T) {
	root := t.TempDir()
	input := []byte("package p\nvar Value = missing\n")
	for name, content := range map[string][]byte{"go.mod": []byte("module example.com/retry\n\ngo 1.27.1\n"),
		"main.tgo": input} {
		if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var failure any
	func() { defer func() { failure = recover() }(); Compile(root) }()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), "undefined: missing") {
		t.Fatalf("source error: %v", failure)
	}
	_, err := os.Stat(filepath.Join(root, "main_tgo.go"))
	if !os.IsNotExist(err) {
		t.Fatalf("failed compilation left output: %v", err)
	}
	corrected := []byte("package p\nvar Value = 3\n")
	if err := os.WriteFile(filepath.Join(root, "main.tgo"), corrected, 0600); err != nil {
		t.Fatal(err)
	}
	Compile(root)
	got, err := os.ReadFile(filepath.Join(root, "main_tgo.go"))
	if err != nil || !bytes.Equal(got, corrected) {
		t.Fatalf("retry output: %v", err)
	}
}

// TestCompileKeepsModuleUpdates checks that module loading retains downloaded dependency checksums.
func TestCompileKeepsModuleUpdates(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOFLAGS", "-mod=mod")
	for name, content := range map[string]string{
		"go.mod":   "module example.com/metadata\n\ngo 1.27.1\n\nrequire golang.org/x/mod v0.42.0\n",
		"main.tgo": "package p\nimport \"golang.org/x/mod/modfile\"\nvar Value = modfile.File{}\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	Compile(root)
	sums, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil || len(sums) == 0 {
		t.Fatalf("module sums: %v", err)
	}
}

// TestCompileKeepsDataAndNestedModuleFiles checks preservation of data folders and nested modules.
func TestCompileKeepsDataAndNestedModuleFiles(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"go.mod":                    []byte("module example.com/outer\n\ngo 1.27.1\n"),
		"main.tgo":                  []byte("package p\nvar Value = if true {1} else {2}\n"),
		"testdata/example.tgo":      []byte("invalid fixture input"),
		"testdata/example_tgo.go":   []byte("keep fixture snapshot"),
		".data/example.tgo":         []byte("invalid hidden input"),
		".data/example_tgo.go":      []byte("keep hidden snapshot"),
		"_data/example.tgo":         []byte("invalid data input"),
		"_data/example_tgo.go":      []byte("keep data snapshot"),
		"nested/go.mod":             []byte("module example.com/nested\n\ngo 1.27.1\n"),
		"nested/main.tgo":           []byte("invalid TGo source in another module"),
		"nested/main_tgo.go":        []byte("keep another module's snapshot"),
		"nested/deeper/go.mod":      []byte("module example.com/deeper\n\ngo 1.27.1\n"),
		"nested/deeper/main.tgo":    []byte("another invalid input"),
		"nested/deeper/main_tgo.go": []byte("keep the deeper snapshot"),
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	Compile(root)
	for name, want := range files {
		if name == "go.mod" || name == "main.tgo" {
			continue
		}
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("changed asset file %s: %v", name, err)
		}
	}
}
