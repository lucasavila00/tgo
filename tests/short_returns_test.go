package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReturnErrorShorthand checks accepted short returns and source diagnostics for rejected forms.
func TestReturnErrorShorthand(t *testing.T) {
	t.Run("Valid", testShortReturnFixtures)
	t.Run("Rejected", testShortReturnErrors)
}

// testShortReturnFixtures checks short-return snapshots and compares their runtime results with Go references.
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

// testShortReturnErrors checks source diagnostics for rejected short-return fixtures.
func testShortReturnErrors(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/return-error-shorthand-rejected")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name(), func(t *testing.T) {
			checkSourceDiagnostic(t, filepath.Join("testdata/return-error-shorthand-rejected", fixture.Name()))
		})
	}
}
