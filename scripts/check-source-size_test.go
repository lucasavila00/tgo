package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceSizeBoundary(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		fail    bool
	}{
		{"empty.go", "", false},
		{"at_limit.go", strings.Repeat("\n", 700), false},
		{"at_limit.tgo", strings.Repeat("\n", 699) + "x", false},
		{"over_limit.go", strings.Repeat("\n", 701), true},
		{"over_limit.tgo", strings.Repeat("\n", 700) + "x", true},
		{"generated_tgo.go", strings.Repeat("\n", 701), false},
		{"generated_tgo_test.go", strings.Repeat("\n", 701), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), test.name)
			if err := os.WriteFile(path, []byte(test.content), 0600); err != nil {
				t.Fatal(err)
			}
			err := checkSourceFiles([]string{path, ""})
			if (err != nil) != test.fail {
				t.Fatalf("error = %v; want failure %t", err, test.fail)
			}
			if test.fail && !strings.Contains(err.Error(), path+": 701 lines; maximum is 700") {
				t.Fatalf("missing file and count: %v", err)
			}
		})
	}
}
