// Command check-function-comments requires comments on named handwritten Go functions.
package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// main reports undocumented functions and exits with failure if the check fails.
func main() {
	paths, err := exec.Command("git", "ls-files", "-z", "--", "*.go").Output()
	if err == nil {
		err = checkFunctionFiles(strings.Split(string(paths), "\x00"))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// checkFunctionFiles checks tracked handwritten Go files, including tests.
// It skips fixture programs and generated code, and returns all missing comments.
func checkFunctionFiles(paths []string) error {
	var failures []error
	for _, path := range paths {
		if skipFunctionFile(path) {
			continue
		}
		positions := token.NewFileSet()
		file, err := parser.ParseFile(positions, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		if ast.IsGenerated(file) {
			continue
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && !hasFunctionComment(function.Doc) {
				position := positions.Position(function.Pos())
				failures = append(failures, fmt.Errorf("%s: function %s needs a comment", position, function.Name))
			}
		}
	}
	return errors.Join(failures...)
}

// skipFunctionFile identifies fixtures and TGo outputs outside the handwritten-source rule.
func skipFunctionFile(path string) bool {
	path = filepath.ToSlash(path)
	return path == "" || strings.Contains("/"+path, "/tests/testdata/") ||
		strings.HasSuffix(path, "_tgo.go") || strings.HasSuffix(path, "_tgo_test.go")
}

// hasFunctionComment accepts attached explanatory text and excludes tool directives.
func hasFunctionComment(comment *ast.CommentGroup) bool {
	if comment == nil {
		return false
	}
	for _, line := range strings.Split(comment.Text(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "nolint") {
			return true
		}
	}
	return false
}
