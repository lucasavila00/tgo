package tests

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValueExpressions(t *testing.T) {
	fixtures, err := os.ReadDir("testdata/value-expressions")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		if !fixture.IsDir() {
			continue
		}
		t.Run(fixture.Name(), func(t *testing.T) {
			checkFeatureFixture(t, filepath.Join("testdata/value-expressions", fixture.Name()))
		})
	}
}
