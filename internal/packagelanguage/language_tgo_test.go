package packagelanguage

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
		want  Language
		err   string
	}{
		{
			name: "TGo",
			files: map[string]string{
				"model.tgo":      "package model\n",
				"model_test.tgo": "package model\n",
				"model_tgo.go":   "package model\n",
			},
			want: TGo,
		},
		{
			name: "Go",
			files: map[string]string{
				"model.go":      "package model\n",
				"model_test.go": "package model\n",
			},
			want: Go,
		},
		{
			name: "mixed production",
			files: map[string]string{
				"model.tgo": "package model\n",
				"helper.go": "package model\n",
			},
			err: "mixes handwritten TGo and Go files: helper.go, model.tgo",
		},
		{
			name: "mixed test",
			files: map[string]string{
				"model.tgo":     "package model\n",
				"model_test.go": "package model\n",
			},
			err: "mixes handwritten TGo and Go files: model.tgo, model_test.go",
		},
		{
			name: "inactive Go",
			files: map[string]string{
				"model.tgo": "package model\n",
				"helper.go": "//go:build never\n\npackage model\n",
			},
			want: TGo,
		},
		{
			name: "active Go",
			files: map[string]string{
				"model.tgo": "package model\n",
				"helper.go": "//go:build active\n\npackage model\n",
			},
			err: "mixes handwritten TGo and Go files: helper.go, model.tgo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			directory := t.TempDir()
			for name, data := range test.files {
				if err := os.WriteFile(filepath.Join(directory, name), []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			buildContext := build.Default
			tags := defaultTags(
				buildContext.GOOS, buildContext.GOARCH, []string{"active"},
			)
			context := Context{
				CgoEnabled: buildContext.CgoEnabled,
				MatchFile: func(path string, _ Language) (bool, error) {
					return matchFile(path, buildContext.GOOS, buildContext.GOARCH, tags)
				},
			}
			got, err := Classify(context, directory)
			if test.err != "" {
				if err == nil || !strings.Contains(err.Error(), test.err) {
					t.Fatalf("error = %v, want %q", err, test.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("language = %v, want %v", got, test.want)
			}
		})
	}
}

func TestClassifyUsesCgoSelection(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	files := map[string]string{
		"model.tgo": "package model\n",
		"helper.go": "package model\n\nimport \"C\"\n",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	buildContext := build.Default
	context := Context{
		CgoEnabled: false,
		MatchFile: func(path string, _ Language) (bool, error) {
			return matchFile(
				path,
				buildContext.GOOS,
				buildContext.GOARCH,
				defaultTags(buildContext.GOOS, buildContext.GOARCH, nil),
			)
		},
	}
	got, err := Classify(context, directory)
	if err != nil {
		t.Fatal(err)
	}
	if got != TGo {
		t.Fatalf("language = %v, want %v", got, TGo)
	}
}

func TestBuildTagsFromGoFlags(t *testing.T) {
	t.Parallel()
	got, err := BuildTagsFromGoFlags(`-mod=readonly -tags 'alpha,beta'`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "beta"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("tags = %v, want %v", got, want)
	}
}
