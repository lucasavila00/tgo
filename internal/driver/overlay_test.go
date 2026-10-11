package driver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareModuleUsesOnlySourceOverlays(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"go.mod":      []byte("module example.com/overlay\n\ngo 1.27.1\n"),
		"main.tgo":    []byte("package overlay\nvar Value = 1\n"),
		"main_tgo.go": []byte("invalid snapshot"),
		"asset.bin":   bytes.Repeat([]byte{42}, 1<<20),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	prepared := prepareModule(readModule(root))
	if prepared.dir != root {
		t.Fatalf("checking directory = %s, want %s", prepared.dir, root)
	}
	if len(prepared.overlay) != 1 || !bytes.Equal(prepared.overlay[filepath.Join(root, "main_tgo.go")], files["main.tgo"]) {
		t.Fatalf("source overlay = %v", prepared.overlay)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("changed %s: %v", name, err)
		}
	}
}

func TestCompileOverlayRemovesTemporaryFiles(t *testing.T) {
	for _, input := range []string{"package p\nvar Value = 1\n", "package p\nvar Value = missing\n"} {
		t.Run(input, func(t *testing.T) {
			root := t.TempDir()
			temporary := t.TempDir()
			t.Setenv("TMPDIR", temporary)
			for name, content := range map[string]string{"go.mod": "module example.com/cleanup\n\ngo 1.27.1\n", "main.tgo": input} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var failure any
			func() { defer func() { failure = recover() }(); Compile(root) }()
			if strings.Contains(input, "missing") != (failure != nil) {
				t.Fatalf("compilation result: %v", failure)
			}
			entries, err := os.ReadDir(temporary)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("temporary files remain: %v", entries)
			}
		})
	}
}

func TestGeneratedOverlayChecksReplacement(t *testing.T) {
	root := t.TempDir()
	source := []byte("package p\nvar Value int = 1\n")
	path := filepath.Join(root, "main.go")
	for name, content := range map[string][]byte{"go.mod": []byte("module example.com/generatedoverlay\n\ngo 1.27.1\n"), "main.go": source} {
		if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var failure any
	func() {
		defer func() { failure = recover() }()
		checkGeneratedOverlay(root, map[string][]byte{path: []byte("package p\nvar Value int = \"wrong\"\n")})
	}()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), "generated Go:") {
		t.Fatalf("generated overlay error: %v", failure)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, source) {
		t.Fatalf("changed source after generated error: %v", err)
	}
}
