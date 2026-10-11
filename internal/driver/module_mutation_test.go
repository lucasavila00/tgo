package driver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompileKeepsModuleFilesWithWritableGoFlags(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOFLAGS", "-mod=mod")
	for name, content := range map[string]string{"go.mod": "module example.com/readonly\n\ngo 1.27.1\n\nrequire golang.org/x/mod v0.42.0\n", "main.tgo": "package p\nimport \"golang.org/x/mod/modfile\"\nvar Value = modfile.File{}\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	Compile(root)
	if _, err := os.Stat(filepath.Join(root, "go.sum")); !os.IsNotExist(err) {
		t.Fatalf("created source go.sum: %v", err)
	}
}
