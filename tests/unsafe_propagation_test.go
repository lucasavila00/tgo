package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestPropagationUnsafeRejected(t *testing.T) {
	entries, err := os.ReadDir("testdata/propagation-rejected")
	if err != nil {
		t.Fatal(err)
	}
	module, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".tgo") {
			continue
		}
		t.Run(strings.TrimSuffix(entry.Name(), ".tgo"), func(t *testing.T) {
			input, err := os.ReadFile(filepath.Join("testdata/propagation-rejected", entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			expected, err := os.ReadFile(filepath.Join("testdata/propagation-rejected",
				strings.TrimSuffix(entry.Name(), ".tgo")+".txt"))
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			writeFiles(t, dir, map[string][]byte{"go.mod": module, entry.Name(): input})
			defer func() {
				actual := recover()
				if actual == nil {
					t.Fatal("expected rejected propagation")
				}
				if fmt.Sprint(actual) != strings.TrimSpace(string(expected)) {
					t.Fatalf("diagnostic: %v, want %s", actual, expected)
				}
			}()
			transpiler.Compile(dir)
		})
	}
}
