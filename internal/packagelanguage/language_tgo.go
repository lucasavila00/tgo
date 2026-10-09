// Package packagelanguage classifies active package source as Go or TGo.
package packagelanguage

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"tgo/internal/outputname"
	"tgo/pkg/syntax"
)

// Language identifies the handwritten source language of a package.
type Language uint8

const (
	// Unknown identifies a directory with no active production source.
	Unknown Language = iota
	// Go identifies a package with handwritten Go production source.
	Go
	// TGo identifies a package with handwritten TGo production source.
	TGo
)

type sourceFile struct {
	name     string
	language Language
	test     bool
}

// Context selects active source files for one build target.
type Context struct {
	CgoEnabled bool
	MatchFile  func(path string, language Language) (bool, error)
}

// BoundaryError reports active handwritten source from both languages.
type BoundaryError struct {
	Package string
	Files   []string
}

// Error returns the stable package boundary diagnostic.
func (e *BoundaryError) Error() string {
	return fmt.Sprintf(
		"package %s mixes handwritten TGo and Go files: %s",
		e.Package, strings.Join(e.Files, ", "),
	)
}

// Classify returns the language of active handwritten source in one directory.
func Classify(context Context, directory string) (Language, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return Unknown, err
	}
	files := make([]sourceFile, 0)
	for _, entry := range entries {
		file, err := activeFile(context, directory, entry)
		if err != nil {
			return Unknown, err
		}
		if file != nil {
			files = append(files, *file)
		}
	}
	return classify(directory, files)
}

func activeFile(
	context Context,
	directory string,
	entry os.DirEntry,
) (*sourceFile, error) {
	if entry.IsDir() {
		return nil, nil
	}
	name := entry.Name()
	language := Unknown
	match := false
	var err error
	switch {
	case strings.HasSuffix(name, ".tgo"):
		language = TGo
		match, err = context.MatchFile(filepath.Join(directory, name), language)
	case strings.HasSuffix(name, ".go") && !outputname.Reserved(name):
		language = Go
		match, err = context.MatchFile(filepath.Join(directory, name), language)
	default:
		return nil, nil
	}
	if err != nil || !match {
		return nil, err
	}
	path := filepath.Join(directory, name)
	importsC, err := fileImportsC(path)
	if err != nil {
		return nil, err
	}
	if importsC && !context.CgoEnabled {
		return nil, nil
	}
	return &sourceFile{
		name: name, language: language,
		test: strings.HasSuffix(name, "_test.go") ||
			strings.HasSuffix(name, "_test.tgo"),
	}, nil
}

func fileImportsC(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	file, err := syntax.ParseFile(token.NewFileSet(), path, data, 0)
	if err != nil {
		return false, err
	}
	for _, item := range file.Imports {
		importPath, err := strconv.Unquote(item.Path.Value)
		if err != nil {
			return false, err
		}
		if importPath == "C" {
			return true, nil
		}
	}
	return false, nil
}

func classify(directory string, files []sourceFile) (Language, error) {
	production := Unknown
	for _, file := range files {
		if file.test {
			continue
		}
		if production == Unknown {
			production = file.language
			continue
		}
		if production != file.language {
			return Unknown, boundaryError(directory, files)
		}
	}
	if production == Unknown {
		return Unknown, nil
	}
	for _, file := range files {
		if file.test && file.language != production {
			return Unknown, boundaryError(directory, files)
		}
	}
	return production, nil
}

func boundaryError(directory string, files []sourceFile) error {
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.name)
	}
	sort.Strings(names)
	return &BoundaryError{Package: filepath.Base(directory), Files: names}
}
