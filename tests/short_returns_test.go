package tests

import (
	"os"
	"path/filepath"
	"testing"
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
			checkSourceDiagnostic(t, filepath.Join("testdata/return-error-shorthand-rejected", fixture.Name()))
		})
	}
}
