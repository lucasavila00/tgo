package driver

import (
	"bytes"
	"os"
	"path/filepath"
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
