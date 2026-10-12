package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/pkg/transpiler"
)

func TestReturnErrorShorthand(t *testing.T) {
	t.Run("Valid", testShortReturnFixtures)
	t.Run("Rejected", testShortReturnErrors)
}

func testShortReturnFixtures(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/return-error-shorthand")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		if !fixture.IsDir() {
			continue
		}
		t.Run(fixture.Name(), func(t *testing.T) {
			checkFeatureFixture(t, filepath.Join("testdata/return-error-shorthand", fixture.Name()))
		})
	}
}

func testShortReturnErrors(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/return-error-shorthand-rejected")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name(), func(t *testing.T) {
			dir := filepath.Join("testdata/return-error-shorthand-rejected", fixture.Name())
			expected, err := os.ReadFile(filepath.Join(dir, "diagnostic.txt"))
			if err != nil {
				t.Fatal(err)
			}
			input := copyFixture(t, dir)
			var failure any
			func() { defer func() { failure = recover() }(); transpiler.Compile(input) }()
			if failure == nil || !strings.Contains(fmt.Sprint(failure), strings.TrimSpace(string(expected))) {
				t.Fatalf("got %v, want %s", failure, expected)
			}
			if !strings.Contains(fmt.Sprint(failure), "main.tgo:") {
				t.Fatalf("missing source position: %v", failure)
			}
		})
	}
}
