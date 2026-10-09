package format_test

import (
	"bytes"
	goformat "go/format"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
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

func TestSourceFormatsCheckedStruct(t *testing.T) {
	t.Parallel()
	input := "package sample\ntype Port struct{number int} checked\n"
	want := "package sample\n\ntype Port struct{ number int } checked\n"
	got, err := format.Source("sample.tgo", []byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", got, want)
	}
}

func TestSourceMatchesGoCorpus(t *testing.T) {
	if !strings.HasPrefix(runtime.Version(), "go1.27.") {
		t.Fatalf("Go corpus needs Go 1.27; got %s", runtime.Version())
	}
	packages := readGoCorpusManifest(t)
	goRoot := goCorpusRoot(t)
	count := 0
	for _, packagePath := range packages {
		count += checkGoFormatPackage(t, goRoot, packagePath)
	}
	if count == 0 {
		t.Fatal("Go corpus has no source files")
	}
	t.Logf("checked %d Go 1.27 source files", count)
}

func TestSourceMatchesFullGoTree(t *testing.T) {
	if os.Getenv("TGO_FULL_GO_FORMAT_CORPUS") != "1" {
		t.Skip("set TGO_FULL_GO_FORMAT_CORPUS=1 in hosted slow CI")
	}
	if !strings.HasPrefix(runtime.Version(), "go1.27.") {
		t.Fatalf("Go corpus needs Go 1.27; got %s", runtime.Version())
	}
	root := filepath.Join(goCorpusRoot(t), "src")
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		checkGoFormatFile(t, path)
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("Go source tree has no source files")
	}
	t.Logf("checked %d Go 1.27 source files", count)
}

func readGoCorpusManifest(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("../../internal/compiler/testdata/go-corpus/packages.txt")
	if err != nil {
		t.Fatal(err)
	}
	packages := []string(nil)
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			packages = append(packages, line)
		}
	}
	return packages
}

func goCorpusRoot(t *testing.T) string {
	t.Helper()
	output, err := exec.Command("go", "env", "GOROOT").CombinedOutput()
	if err != nil {
		t.Fatalf("find GOROOT: %v\n%s", err, output)
	}
	return strings.TrimSpace(string(output))
}

func checkGoFormatPackage(t *testing.T, goRoot string, packagePath string) int {
	t.Helper()
	directory := filepath.Join(goRoot, "src", filepath.FromSlash(packagePath))
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		checkGoFormatFile(t, filepath.Join(directory, entry.Name()))
		count++
	}
	return count
}

func checkGoFormatFile(t *testing.T, path string) {
	t.Helper()
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
			again, err := format.Source(inputPath, got)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(again, got) {
				t.Fatalf("second pass changed output:\n%s", again)
			}
		})
	}
}

func TestSourceKeepsNestedLineDirectiveActive(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("testdata/directives.input.tgo")
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := format.Source("directives.tgo", source)
	if err != nil {
		t.Fatal(err)
	}
	files := token.NewFileSet()
	file := files.AddFile("directives.tgo", -1, len(formatted))
	lexer := *new(scanner.Scanner)
	lexer.Init(file, formatted, nil, scanner.ScanComments)
	for {
		position, kind, _ := lexer.Scan()
		if kind == token.RETURN {
			got := files.Position(position)
			if got.Filename != "nested.tgo" || got.Line != 200 {
				t.Fatalf("return position = %s, want nested.tgo:200", got)
			}
			return
		}
		if kind == token.EOF {
			t.Fatal("formatted source has no return statement")
		}
	}
}

func TestSourceRepositoryCorpus(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	files := repositoryTGoFiles(t, root)
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
			checkRepositorySource(t, path, name)
		})
	}
}

func repositoryTGoFiles(t *testing.T, root string) []string {
	t.Helper()
	files := make([]string, 0)
	for _, directory := range []string{"cmd", "internal", "pkg"} {
		rootDirectory := filepath.Join(root, directory)
		err := filepath.WalkDir(rootDirectory, func(
			path string,
			entry fs.DirEntry,
			err error,
		) error {
			return collectTGoFile(path, entry, err, &files)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return files
}

func collectTGoFile(path string, entry fs.DirEntry, err error, files *[]string) error {
	if err != nil {
		return err
	}
	if entry.IsDir() && entry.Name() == "testdata" {
		return filepath.SkipDir
	}
	if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tgo") {
		*files = append(*files, path)
	}
	return nil
}

func checkRepositorySource(t *testing.T, path string, name string) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := format.Source(path, source)
	if err != nil {
		t.Fatal(err)
	}
	if (strings.HasPrefix(name, "pkg/format/") || name == "cmd/tgofmt/main.tgo") &&
		!bytes.Equal(formatted, source) {
		t.Fatal("formatter source is not in canonical format")
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
