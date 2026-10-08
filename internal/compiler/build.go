package compiler

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Build checks tgo packages and writes Go source beside each input file.
// If a check fails, it restores all output files changed by this build.
func Build(directory string, patterns []string) (err error) {
	root, module, err := moduleRoot(directory)
	if err != nil {
		return err
	}
	packages, err := discover(root, module)
	if err != nil {
		return err
	}
	selected, err := selectPackages(directory, patterns, packages)
	if err != nil {
		return err
	}
	builder := packageBuilder{
		packages: packages,
		states:   make(map[string]buildState),
		previous: make(map[string]previousFile),
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, builder.restore())
		}
	}()
	for _, path := range selected {
		if err = builder.build(path); err != nil {
			return err
		}
	}
	return nil
}

func moduleRoot(directory string) (string, string, error) {
	command := exec.Command("go", "env", "GOMOD")
	command.Dir = directory
	output, err := command.Output()
	if err != nil {
		return "", "", fmt.Errorf("find go.mod: %w", err)
	}
	path := strings.TrimSpace(string(output))
	if path == "" || path == os.DevNull {
		return "", "", errors.New("tgo needs a Go module; run go mod init first")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return filepath.Dir(path), strings.Trim(fields[1], "\""), nil
		}
	}
	return "", "", errors.New("go.mod has no module path")
}

func discover(root, module string) (map[string]*packageUnit, error) {
	discovery := packageDiscovery{
		root:     root,
		module:   module,
		packages: make(map[string]*packageUnit),
	}
	err := filepath.WalkDir(root, discovery.visit)
	if err != nil {
		return nil, err
	}
	for _, unit := range discovery.packages {
		unit.Imports = discovery.packages
		if err := unit.readGoFiles(); err != nil {
			return nil, err
		}
	}
	return discovery.packages, nil
}

type packageDiscovery struct {
	root     string
	module   string
	packages map[string]*packageUnit
}

func (d *packageDiscovery) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() {
		return d.visitDirectory(path, entry.Name())
	}
	if filepath.Ext(path) != ".tgo" {
		return nil
	}
	return d.addSource(path)
}

func (d *packageDiscovery) visitDirectory(path, name string) error {
	if path == d.root {
		return nil
	}
	if strings.HasPrefix(name, ".") || name == "vendor" {
		return filepath.SkipDir
	}
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return filepath.SkipDir
	}
	return nil
}

func (d *packageDiscovery) addSource(path string) error {
	directory := filepath.Dir(path)
	importPath, err := d.importPath(directory)
	if err != nil {
		return err
	}
	unit := d.packages[importPath]
	if unit == nil {
		unit = &packageUnit{
			Dir:    directory,
			Path:   importPath,
			Models: make(map[string]*model),
			fs:     token.NewFileSet(),
		}
		d.packages[importPath] = unit
	}
	return unit.readSource(path)
}

func (d *packageDiscovery) importPath(directory string) (string, error) {
	relative, err := filepath.Rel(d.root, directory)
	if err != nil {
		return "", err
	}
	if relative == "." {
		return d.module, nil
	}
	return d.module + "/" + filepath.ToSlash(relative), nil
}

func (p *packageUnit) readSource(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	source, err := parseSource(p.fs, path, data)
	if err != nil {
		return err
	}
	for _, model := range source.Models {
		if p.Models[model.Name] != nil {
			return fmt.Errorf("%s: duplicate type %s", path, model.Name)
		}
		p.Models[model.Name] = model
	}
	p.Sources = append(p.Sources, source)
	p.Files = append(p.Files, source.File)
	return nil
}

func (p *packageUnit) readGoFiles() error {
	entries, err := os.ReadDir(p.Dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if strings.HasSuffix(name, "_tgo.go") {
			continue
		}
		matches, err := build.Default.MatchFile(p.Dir, name)
		if err != nil {
			return err
		}
		if !matches {
			continue
		}
		mode := parser.ParseComments | parser.SkipObjectResolution
		file, err := parser.ParseFile(p.fs, filepath.Join(p.Dir, name), nil, mode)
		if err != nil {
			return err
		}
		p.Files = append(p.Files, file)
	}
	return nil
}

func selectPackages(
	directory string,
	patterns []string,
	packages map[string]*packageUnit,
) ([]string, error) {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	selected := make(map[string]bool)
	for _, pattern := range patterns {
		target, recursive := packagePattern(directory, pattern)
		matched := false
		for path, unit := range packages {
			if packageMatches(unit, path, pattern, target, recursive) {
				selected[path] = true
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("%s: no tgo packages", pattern)
		}
	}
	paths := make([]string, 0, len(selected))
	for path := range selected {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func packagePattern(directory, pattern string) (string, bool) {
	recursive := strings.HasSuffix(pattern, "/...") || pattern == "..."
	target := strings.TrimSuffix(pattern, "/...")
	if target == "..." {
		target = "."
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(directory, target)
	}
	return filepath.Clean(target), recursive
}

func packageMatches(
	unit *packageUnit,
	path string,
	pattern string,
	target string,
	recursive bool,
) bool {
	if path == pattern || unit.Dir == target {
		return true
	}
	childPrefix := target + string(filepath.Separator)
	return recursive && strings.HasPrefix(unit.Dir, childPrefix)
}

type buildState uint8

const (
	buildNew buildState = iota
	buildActive
	buildDone
)

type previousFile struct {
	data   []byte
	exists bool
}

type packageBuilder struct {
	packages map[string]*packageUnit
	states   map[string]buildState
	previous map[string]previousFile
}

func (b *packageBuilder) build(path string) error {
	switch b.states[path] {
	case buildDone:
		return nil
	case buildActive:
		return fmt.Errorf("import cycle at %s", path)
	}
	b.states[path] = buildActive
	unit := b.packages[path]
	for _, dependency := range importsOf(unit.Files) {
		if b.packages[dependency] == nil {
			continue
		}
		if err := b.build(dependency); err != nil {
			return err
		}
	}
	outputs, err := unit.compile()
	if err != nil {
		return err
	}
	for name, data := range outputs {
		if err := b.write(name, data); err != nil {
			return err
		}
	}
	b.states[path] = buildDone
	return nil
}

func importsOf(files []*ast.File) []string {
	imports := make(map[string]bool)
	for _, file := range files {
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err == nil {
				imports[path] = true
			}
		}
	}
	paths := make([]string, 0, len(imports))
	for path := range imports {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func (b *packageBuilder) write(path string, data []byte) error {
	previous, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	exists := err == nil
	if exists && !strings.HasPrefix(string(previous), "// Code generated by tgo. DO NOT EDIT.") {
		return fmt.Errorf("refusing to replace user file %s", path)
	}
	b.previous[path] = previousFile{data: previous, exists: exists}
	return os.WriteFile(path, data, 0o644)
}

func (b *packageBuilder) restore() error {
	var failures []error
	for path, previous := range b.previous {
		var err error
		if previous.exists {
			err = os.WriteFile(path, previous.data, 0o644)
		} else {
			err = os.Remove(path)
		}
		if err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
