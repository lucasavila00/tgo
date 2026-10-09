package driver

import (
	"context"
	"errors"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"tgo/internal/outputname"
	"tgo/internal/packagelanguage"
)

func TestRepositoryTGoPackagesUseTGoTests(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	context, err := effectiveBuildContext(root)
	if err != nil {
		t.Fatal(err)
	}
	modules, err := repositoryModules(root)
	if err != nil {
		t.Fatal(err)
	}
	manualTests := []string{}
	for _, module := range modules {
		found, err := moduleGoTestsInTGoPackages(module, &context)
		if err != nil {
			t.Fatal(err)
		}
		manualTests = append(
			manualTests,
			found...,
		)
	}
	sort.Strings(manualTests)
	for _, path := range manualTests {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("TGo package has handwritten Go test: %s", relative)
	}
}

func TestRepositoryPackagesHaveOneLanguage(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	context, err := effectiveBuildContext(root)
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
				name == "testdata" || name == "vendor" {
				return filepath.SkipDir
			}
		}
		_, err = packagelanguage.Classify(
			packagelanguage.ContextFromBuild(&context), path,
		)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBuildRejectsMixedPackage(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackageLanguageFile(t, root, "go.mod", "module example.com/mixed\n\ngo 1.27\n")
	directory := filepath.Join(root, "app")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	writePackageLanguageFile(t, directory, "app.tgo", "package app\n")
	writePackageLanguageFile(t, directory, "helper.go", "package app\n")
	err := Build(root, []string{"./app"})
	want := "package app mixes handwritten TGo and Go files: app.tgo, helper.go"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
	_, err = CompileAvailableWorkspaceContext(context.Background(), root)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("workspace error = %v, want %q", err, want)
	}
	_, err = CompileAvailableWorkspaceViewsContext(context.Background(), root)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("workspace views error = %v, want %q", err, want)
	}
	if _, err := os.Stat(filepath.Join(directory, "app_tgo.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("generated output exists after rejection: %v", err)
	}
}

func TestBuildRejectsTestLanguageMismatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		production string
		test       string
	}{
		{name: "TGo package", production: "app.tgo", test: "app_test.go"},
		{name: "Go package", production: "app.go", test: "app_test.tgo"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writePackageLanguageFile(
				t, root, "go.mod", "module example.com/mixedtest\n\ngo 1.27\n",
			)
			directory := filepath.Join(root, "app")
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			writePackageLanguageFile(t, directory, test.production, "package app\n")
			writePackageLanguageFile(t, directory, test.test, "package app\n")
			err := Build(root, []string{"./app"})
			want := "package app mixes handwritten TGo and Go files"
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestBuildIgnoresInactiveOtherLanguage(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePackageLanguageFile(t, root, "go.mod", "module example.com/target\n\ngo 1.27\n")
	directory := filepath.Join(root, "app")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	writePackageLanguageFile(t, directory, "app.tgo", "package app\n")
	writePackageLanguageFile(
		t,
		directory,
		"helper.go",
		"//go:build tgo_inactive_target\n\npackage app\n",
	)
	if err := Build(root, []string{"./app"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "app_tgo.go")); err != nil {
		t.Fatal(err)
	}
}

func TestMatchingTestSourcesIgnoresCgoWhenDisabled(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "app_test.tgo")
	writePackageLanguageFile(
		t,
		directory,
		"app_test.tgo",
		"package app\n\nimport \"C\"\n",
	)
	buildContext := build.Default
	buildContext.CgoEnabled = false
	unit := packageUnit{
		Dir:             directory,
		testSourcePaths: []string{path},
		context:         &buildContext,
	}
	paths, err := unit.matchingTestSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("matching test sources = %v, want none", paths)
	}
}

func writePackageLanguageFile(t *testing.T, directory, name, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func repositoryModules(root string) ([]string, error) {
	modules := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path != root {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
				name == "vendor" {
				return filepath.SkipDir
			}
		}
		if entry.Name() == "go.mod" {
			modules = append(modules, filepath.Dir(path))
		}
		return nil
	})
	sort.Strings(modules)
	return modules, err
}

func moduleGoTestsInTGoPackages(
	root string,
	context *build.Context,
) ([]string, error) {
	manualTests := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == root {
				return nil
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
				name == "testdata" || name == "vendor" {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		}
		directory := filepath.Dir(path)
		if !strings.HasSuffix(entry.Name(), ".tgo") ||
			strings.HasSuffix(entry.Name(), "_test.tgo") {
			return nil
		}
		matches, err := packagelanguage.MatchFile(
			context, path, packagelanguage.TGo,
		)
		if err != nil || !matches {
			return err
		}
		entries, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		for _, candidate := range entries {
			name := candidate.Name()
			if !candidate.IsDir() && strings.HasSuffix(name, "_test.go") &&
				!outputname.Reserved(name) {
				manualTests = append(manualTests, filepath.Join(directory, name))
			}
		}
		return filepath.SkipDir
	})
	return manualTests, err
}
