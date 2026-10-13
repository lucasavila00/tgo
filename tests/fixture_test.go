package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

// compiledFixture keeps the saved fixture separate from its compiled work copy.
type compiledFixture struct {
	directory string
	original  map[string][]byte
	generated map[string][]byte
}

func compileFixture(t *testing.T, directory string) compiledFixture {
	t.Helper()
	original := readFiles(t, directory)
	work := t.TempDir()
	writeFiles(t, work, original)
	transpiler.Compile(work)
	return compiledFixture{directory: work, original: original, generated: readFiles(t, work)}
}

// checkSourceDiagnostic checks the expected message and its TGo source position.
func checkSourceDiagnostic(t *testing.T, directory string) {
	t.Helper()
	expected, err := os.ReadFile(filepath.Join(directory, "diagnostic.txt"))
	if err != nil {
		t.Fatal(err)
	}
	work := copyFixture(t, directory)
	var failure any
	func() {
		defer func() { failure = recover() }()
		transpiler.Compile(work)
	}()
	message := fmt.Sprint(failure)
	if failure == nil || !strings.Contains(message, strings.TrimSpace(string(expected))) {
		t.Fatalf("got %v, want %s", failure, expected)
	}
	if !strings.Contains(message, "main.tgo:") {
		t.Fatalf("missing source position: %v", failure)
	}
}
