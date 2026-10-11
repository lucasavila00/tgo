package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedGoErrors(t *testing.T) {
	for _, test := range []struct{ name, file, source string }{
		{"ordinary source", "main.go", "package p\nvar Value int = \"wrong\"\n"},
		{"test variant", "main_test.go", "package p_test\nvar Value int = \"wrong\"\n"},
		{"dependency", "dep/main.go", "package dep\nvar Value int = \"wrong\"\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			files := map[string]string{"go.mod": "module example.com/generated\n\ngo 1.27.1\n", "main.go": "package p\n"}
			if test.name == "dependency" {
				files["main.go"] = "package p\nimport \"example.com/generated/dep\"\nvar Value = dep.Value\n"
			}
			files[test.file] = test.source
			for name, source := range files {
				path := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			defer func() {
				err := recover()
				if err == nil || !strings.Contains(fmt.Sprint(err), "cannot use") {
					t.Fatalf("generated Go diagnostic: %v", err)
				}
			}()
			checkGenerated(dir)
		})
	}
}
