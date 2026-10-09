package driver

import (
	"bytes"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"tgo/internal/outputname"
)

func TestGoCorpusIdentity(t *testing.T) {
	if !strings.HasPrefix(runtime.Version(), "go1.27.") {
		t.Fatalf("Go corpus needs Go 1.27; got %s", runtime.Version())
	}
	packages := readGoCorpusManifest(t)
	goRoot := goCorpusRoot(t)
	root := t.TempDir()
	writeIdentityModule(t, root)
	sources := make(map[string][]byte)
	testFiles := 0
	externalTests := 0
	for _, packagePath := range packages {
		included, excluded := copyGoCorpusPackage(t, goRoot, root, packagePath, sources)
		testFiles += included
		externalTests += excluded
	}
	if err := Build(root, []string{"./generated/..."}); err != nil {
		t.Fatal(err)
	}
	active := compareGoCorpusOutput(t, goRoot, root, sources)
	runGoCorpusTests(t, root, "./original/...", "./generated/...")
	t.Logf(
		"%s: %d packages, %d active source files, "+
			"%d same-package test files, %d external test files excluded",
		runtime.Version(), len(packages), active, testFiles, externalTests,
	)
}

func goCorpusRoot(t *testing.T) string {
	t.Helper()
	command := exec.Command("go", "env", "GOROOT")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("find GOROOT: %v\n%s", err, output)
	}
	return strings.TrimSpace(string(output))
}

func readGoCorpusManifest(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("testdata/go-corpus/packages.txt")
	if err != nil {
		t.Fatal(err)
	}
	var packages []string
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			packages = append(packages, line)
		}
	}
	if len(packages) == 0 {
		t.Fatal("Go corpus manifest is empty")
	}
	return packages
}

func writeIdentityModule(t *testing.T, root string) {
	t.Helper()
	data := []byte("module identity.test\n\ngo 1.27.0\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func copyGoCorpusPackage(
	t *testing.T,
	goRoot string,
	root string,
	packagePath string,
	sources map[string][]byte,
) (includedTests, excludedTests int) {
	t.Helper()
	sourceDirectory := filepath.Join(goRoot, "src", filepath.FromSlash(packagePath))
	entries, err := os.ReadDir(sourceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	packageName := ""
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sourceDirectory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if packageName == "" {
			packageName = sourcePackageName(t, entry.Name(), data)
		}
		copyCorpusFile(t, root, "original", packagePath, entry.Name(), data)
		tgoName := strings.TrimSuffix(entry.Name(), ".go") + ".tgo"
		copyCorpusFile(t, root, "generated", packagePath, tgoName, data)
		sources[filepath.Join(packagePath, tgoName)] = data
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sourceDirectory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if sourcePackageName(t, entry.Name(), data) != packageName {
			excludedTests++
			continue
		}
		copyCorpusFile(t, root, "original", packagePath, entry.Name(), data)
		copyCorpusFile(t, root, "generated", packagePath, entry.Name(), data)
		includedTests++
	}
	if packageName == "" {
		t.Fatalf("%s has no Go source", packagePath)
	}
	return includedTests, excludedTests
}

func sourcePackageName(t *testing.T, name string, data []byte) string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, data, parser.PackageClauseOnly)
	if err != nil {
		t.Fatal(err)
	}
	return file.Name.Name
}

func copyCorpusFile(
	t *testing.T,
	root string,
	kind string,
	packagePath string,
	name string,
	data []byte,
) {
	t.Helper()
	directory := filepath.Join(root, kind, filepath.FromSlash(packagePath))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func compareGoCorpusOutput(
	t *testing.T,
	goRoot string,
	root string,
	sources map[string][]byte,
) int {
	t.Helper()
	paths := make([]string, 0, len(sources))
	for path := range sources {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	active := 0
	for _, sourcePath := range paths {
		directory := filepath.Join(root, "generated", filepath.Dir(sourcePath))
		name := filepath.Base(sourcePath)
		goName := strings.TrimSuffix(name, ".tgo") + ".go"
		sourceDirectory := filepath.Join(goRoot, "src", filepath.Dir(sourcePath))
		match, err := build.Default.MatchFile(sourceDirectory, goName)
		if err != nil {
			t.Fatal(err)
		}
		output := filepath.Join(directory, filepath.Base(outputname.Path(name)))
		if !match {
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("inactive source emitted %s", output)
			}
			continue
		}
		active++
		want := sources[sourcePath]
		got, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s changed ordinary Go\n%s", sourcePath, lineDiff(want, got))
		}
	}
	return active
}

func runGoCorpusTests(t *testing.T, root string, patterns ...string) {
	t.Helper()
	for _, pattern := range patterns {
		command := exec.Command("go", "test", pattern)
		command.Dir = root
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", strconv.Quote(pattern), err, output)
		}
	}
}
