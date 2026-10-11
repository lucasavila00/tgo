package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestTypedTemporaryRejected(t *testing.T) {
	entries, err := os.ReadDir("testdata/typed-temporaries-rejected")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			dir := filepath.Join("testdata/typed-temporaries-rejected", entry.Name())
			expected, err := os.ReadFile(filepath.Join(dir, "diagnostic.txt"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				actual := recover()
				if fmt.Sprint(actual) != strings.TrimSpace(string(expected)) {
					t.Fatalf("diagnostic: %v, want %s", actual, expected)
				}
			}()
			transpiler.Compile(dir)
		})
	}
}
