package driver

import (
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
		matches, err := matchTgoFile(context, path)
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
