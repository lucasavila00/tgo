package outputname

import (
	"path/filepath"
	"testing"
)

func TestPath(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{source: "model.tgo", want: "model_tgo.go"},
		{source: "model_linux.tgo", want: "model_tgo_linux.go"},
		{source: "model_amd64.tgo", want: "model_tgo_amd64.go"},
		{source: "model_linux_amd64.tgo", want: "model_tgo_linux_amd64.go"},
		{source: "foo_tgo_bar_linux.tgo", want: "foo_tgo_bar_tgo_linux.go"},
		{source: "model_hack.tgo", want: "model_hack_tgo.go"},
		{
			source: filepath.Join("dir", "model_linux.tgo"),
			want:   filepath.Join("dir", "model_tgo_linux.go"),
		},
	}
	for _, test := range tests {
		if got := Path(test.source); got != test.want {
			t.Errorf("Path(%q) = %q, want %q", test.source, got, test.want)
		}
	}
}

func TestMatches(t *testing.T) {
	tests := []struct {
		source    string
		generated string
		want      bool
	}{
		{source: "model.tgo", generated: "/work/model_tgo.go", want: true},
		{
			source:    "foo_tgo_bar_linux.tgo",
			generated: "/work/foo_tgo_bar_tgo_linux.go",
			want:      true,
		},
		{source: "model_hack.tgo", generated: "/work/model_hack_tgo.go", want: true},
		{source: "model_hack.tgo", generated: "/work/model_tgo_hack.go", want: false},
		{source: "model.go", generated: "/work/model_tgo.go", want: false},
		{source: "dir/model.tgo", generated: "/work/model_tgo.go", want: false},
	}
	for _, test := range tests {
		if got := Matches(test.source, test.generated); got != test.want {
			t.Errorf(
				"Matches(%q, %q) = %v, want %v",
				test.source,
				test.generated,
				got,
				test.want,
			)
		}
	}
}

func TestReserved(t *testing.T) {
	for _, name := range []string{"model_tgo.go", "model_tgo_linux.go"} {
		if !Reserved(name) {
			t.Errorf("Reserved(%q) = false", name)
		}
	}
	for _, name := range []string{"model.go", "model_tgo.txt"} {
		if Reserved(name) {
			t.Errorf("Reserved(%q) = true", name)
		}
	}
}
