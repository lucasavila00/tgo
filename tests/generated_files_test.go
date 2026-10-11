package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestGeneratedFilesIgnoreSnapshots(t *testing.T) {
	dir := t.TempDir()
	target := "selected_" + runtime.GOOS + "_" + runtime.GOARCH
	files := map[string][]byte{
		"go.mod":             []byte("module example.com/generated\n\ngo 1.27.1\n"),
		"main.tgo":           []byte("package example\nfunc Value() int { return targetValue }\n"),
		target + ".tgo":      []byte("package example\nconst targetValue = 7\n"),
		target + "_test.tgo": []byte("package example\nimport \"testing\"\nfunc TestValue(t *testing.T) { if Value()!=7 { t.Fatal(Value()) } }\n"),
		"main_tgo.go":        []byte("this stale snapshot must not be parsed"),
	}
	excludedOS := "windows"
	if runtime.GOOS == excludedOS {
		excludedOS = "linux"
	}
	excluded := "excluded_" + excludedOS + ".tgo"
	files[excluded] = []byte("package example\nvar notForThisTarget = missingName\n")
	files["excluded_tgo_"+excludedOS+".go"] = []byte("invalid stale snapshot")

	writeFiles(t, dir, files)
	transpiler.Compile(dir)
	first := readFiles(t, dir)
	if _, err := os.Stat(filepath.Join(dir, "excluded_tgo_"+excludedOS+".go")); !os.IsNotExist(err) {
		t.Fatalf("excluded generated file remains: %v", err)
	}
	for _, name := range []string{"main_tgo.go", "selected_tgo_" + runtime.GOOS + "_" + runtime.GOARCH + ".go", "selected_tgo_" + runtime.GOOS + "_" + runtime.GOARCH + "_test.go"} {
		if len(first[name]) == 0 {
			t.Fatalf("missing %s", name)
		}
	}
	for name, content := range files {
		if filepath.Ext(name) == ".tgo" && !bytes.Equal(first[name], content) {
			t.Fatalf("source changed: %s", name)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "main_tgo.go"), []byte("package example\nvar _tgoValue0 = nonexistent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	transpiler.Compile(dir)
	second := readFiles(t, dir)
	for name, content := range first {
		if !bytes.Equal(second[name], content) {
			t.Fatalf("snapshot changed compilation of %s", name)
		}
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = dir
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated tests: %v\n%s", err, data)
	}
}
