package format_test

import (
	"bytes"
	goformat "go/format"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tgo/pkg/format"
	"tgo/pkg/syntax"
)

func TestSourceBasic(t *testing.T) {
	t.Parallel()
	input := "package sample\nfunc add(left int,right int)int{return left+right}\n"
	want := "package sample\n\nfunc add(left int, right int) int { return left + right }\n"
	got, err := format.Source("sample.tgo", []byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
	again, err := format.Source("sample.tgo", got)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(got) {
		t.Fatalf("second pass changed output:\n%s", again)
	}
}

func TestSourceMatchesGoCorpus(t *testing.T) {
	if !strings.HasPrefix(runtime.Version(), "go1.27.") {
		t.Fatalf("Go corpus needs Go 1.27; got %s", runtime.Version())
	}
	manifest, err := os.ReadFile("../../internal/compiler/testdata/go-corpus/packages.txt")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for line := range strings.SplitSeq(string(manifest), "\n") {
		packagePath := strings.TrimSpace(line)
		if packagePath == "" || strings.HasPrefix(packagePath, "#") {
			continue
		}
		directory := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(packagePath))
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want, err := goformat.Source(source)
			if err != nil {
				t.Fatal(err)
			}
			got, err := format.Source(path, source)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s differs from Go format", path)
			}
			count++
		}
	}
	if count == 0 {
		t.Fatal("Go corpus has no source files")
	}
	t.Logf("checked %d Go 1.27 source files", count)
}

func TestSourceMatchesGoFormatForOrdinarySyntax(t *testing.T) {
	t.Parallel()
	inputs, err := filepath.Glob("testdata/go/*.input.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, inputPath := range inputs {
		inputPath := inputPath
		name := strings.TrimSuffix(filepath.Base(inputPath), ".input.go")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatal(err)
			}
			want, err := goformat.Source(input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := format.Source(inputPath, input)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("formatted source:\n%s\nwant Go format:\n%s", got, want)
			}
		})
	}
}

func TestSourceRepositoryCorpus(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	files := make([]string, 0)
	for _, directory := range []string{"cmd", "internal", "pkg"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() && entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tgo") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(files) == 0 {
		t.Fatal("repository corpus has no TGo source")
	}
	for _, path := range files {
		path := path
		name, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			formatted, err := format.Source(path, source)
			if err != nil {
				t.Fatal(err)
			}
			again, err := format.Source(path, formatted)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(again, formatted) {
				t.Fatal("second formatting pass changed output")
			}
			before := commentTexts(t, path, source)
			after := commentTexts(t, path, formatted)
			if !equalStrings(before, after) {
				t.Fatalf("comments changed:\n%q\nwant:\n%q", after, before)
			}
		})
	}
}

func commentTexts(t *testing.T, filename string, source []byte) []string {
	t.Helper()
	file, err := syntax.ParseFile(
		token.NewFileSet(),
		filename,
		source,
		syntax.ParseComments|syntax.AllErrors|syntax.AllowInvalidModels,
	)
	if err != nil {
		t.Fatal(err)
	}
	comments := make([]string, 0)
	for _, group := range file.Comments {
		for _, comment := range group.List {
			comments = append(comments, comment.Text)
		}
	}
	return comments
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestSourceFixtures(t *testing.T) {
	t.Parallel()
	inputs, err := filepath.Glob("testdata/*.input.tgo")
	if err != nil {
		t.Fatal(err)
	}
	for _, inputPath := range inputs {
		inputPath := inputPath
		name := strings.TrimSuffix(filepath.Base(inputPath), ".input.tgo")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatal(err)
			}
			wantPath := strings.TrimSuffix(inputPath, ".input.tgo") + ".golden.tgo"
			want, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatal(err)
			}
			got, err := format.Source(inputPath, input)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
			}
			again, err := format.Source(wantPath, got)
			if err != nil {
				t.Fatal(err)
			}
			if string(again) != string(got) {
				t.Fatalf("second pass changed output:\n%s", again)
			}
		})
	}
}

func TestSourceRejectsMalformedInput(t *testing.T) {
	t.Parallel()
	_, err := format.Source("bad.tgo", []byte("package sample\nfunc {"))
	if err == nil {
		t.Fatal("format accepted malformed input")
	}
}
