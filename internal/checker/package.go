package checker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lucasavila00/tgo/internal/checker/gcimporter"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
	"golang.org/x/tools/go/packages"
)

// Package keeps checked syntax and private type objects together.
type Package struct {
	Fset           *token.FileSet
	Syntax         []*ast.File
	Types          *types.Package
	TypesInfo      *types.Info
	GoFiles        []string
	Errors         []error
	Operations     map[string]types.TGoOperation
	Comprehensions map[*ast.CallExpr]*frontend.Comprehension
	Conditionals   map[*ast.CallExpr]ast.Stmt
}

// Loader checks source packages and reads compiled dependency types.
// It shares private type objects across those checks and imports.
type Loader struct {
	Root     string
	TGoFiles map[string]string
	Used     map[string]bool
	packages map[string]*Package
	checking map[string]bool
	imports  map[string]*types.Package
	exports  map[string]string
	fset     *token.FileSet
	sizes    types.Sizes
}

// NewLoader records dependency exports before any source package is checked.
// Every source check uses the module's selected Go target architecture.
func NewLoader(root string, roots []*packages.Package, tgoFiles map[string]string, used map[string]bool) (*Loader, error) {
	sizes, err := targetSizes(root)
	if err != nil {
		return nil, err
	}
	loader := &Loader{Root: root, TGoFiles: tgoFiles, Used: used,
		packages: make(map[string]*Package), checking: make(map[string]bool),
		imports: make(map[string]*types.Package), exports: make(map[string]string), fset: token.NewFileSet(), sizes: sizes}
	packages.Visit(roots, nil, func(pkg *packages.Package) {
		if pkg.ExportFile != "" {
			loader.exports[pkg.PkgPath] = pkg.ExportFile
		}
	})
	return loader, nil
}

func newInfo() *types.Info {
	return &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Implicits: make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection), Scopes: make(map[ast.Node]*types.Scope),
		Instances: make(map[*ast.Ident]types.Instance), FileVersions: make(map[*ast.File]string),
	}
}

// Check parses and checks one selected source package and its dependencies.
// A successful result is cached by package ID; test variants have separate IDs.
func (loader *Loader) Check(metadata *packages.Package) (*Package, error) {
	if pkg := loader.packages[metadata.ID]; pkg != nil {
		return pkg, nil
	}
	if loader.checking[metadata.ID] {
		return nil, fmt.Errorf("import cycle through %s", metadata.PkgPath)
	}
	loader.checking[metadata.ID] = true
	defer delete(loader.checking, metadata.ID)
	pkg := &Package{
		Fset:           loader.fset,
		TypesInfo:      newInfo(),
		GoFiles:        metadata.GoFiles,
		Operations:     make(map[string]types.TGoOperation),
		Comprehensions: make(map[*ast.CallExpr]*frontend.Comprehension),
		Conditionals:   make(map[*ast.CallExpr]ast.Stmt),
	}
	filenames := metadata.CompiledGoFiles
	if len(filenames) == 0 {
		filenames = metadata.GoFiles
	}
	for _, filename := range filenames {
		input, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		name, err := filepath.Rel(loader.Root, filename)
		if err != nil {
			return nil, err
		}
		var file *ast.File
		if sourceName := loader.TGoFiles[name]; sourceName != "" {
			parsed, err := frontend.ParseFile(loader.fset, filename, sourceName, input, loader.Used)
			if err != nil {
				return nil, err
			}
			file = parsed.File
			for name, record := range parsed.Comprehensions {
				pkg.Operations[name] = types.TGoOperation{Comprehension: true, Ranges: record.Ranges, Output: record.Output, Position: record.Position}
				pkg.Operations[record.Output] = types.TGoOperation{Element: true, Position: record.Position}
				ast.Inspect(file, func(node ast.Node) bool {
					if call, ok := node.(*ast.CallExpr); ok {
						if id, ok := call.Fun.(*ast.Ident); ok && id.Name == name {
							pkg.Comprehensions[call] = record
						}
					}
					return true
				})
			}
			for name, pos := range parsed.Conditionals {
				pkg.Operations[name] = types.TGoOperation{Conditional: true, Position: pos}
			}
			for name, op := range parsed.Propagations {
				pkg.Operations[name] = types.TGoOperation{Wrap: op.Wrap, Position: op.Position, CallName: op.CallName}
			}
		} else {
			file, err = parser.ParseFile(loader.fset, filename, input, parser.ParseComments|parser.SkipObjectResolution)
			if err != nil {
				return nil, err
			}
		}
		pkg.Syntax = append(pkg.Syntax, file)
	}
	return loader.check(metadata, pkg)
}

func (loader *Loader) check(metadata *packages.Package, pkg *Package) (*Package, error) {
	config := types.Config{TGo: pkg.Operations, Importer: packageImporter{loader, metadata}, Sizes: loader.sizes, Error: func(err error) { pkg.Errors = append(pkg.Errors, err) }}
	if metadata.Module != nil {
		config.GoVersion = "go" + metadata.Module.GoVersion
	}
	checkedTypes, err := config.Check(metadata.PkgPath, pkg.Fset, pkg.Syntax, pkg.TypesInfo)
	pkg.Types = checkedTypes
	if err != nil {
		return nil, err
	}
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && pkg.Operations[id.Name].Conditional {
					pkg.Conditionals[call] = call.Args[0].(*ast.FuncLit).Body.List[0]
				}
			}
			return true
		})
	}
	loader.packages[metadata.ID] = pkg
	return pkg, nil
}

func targetSizes(root string) (types.Sizes, error) {
	command := exec.Command("go", "env", "GOARCH")
	command.Dir = root
	arch, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("read Go target architecture: %w", err)
	}
	return types.SizesFor("gc", strings.TrimSpace(string(arch))), nil
}

type packageImporter struct {
	loader   *Loader
	metadata *packages.Package
}

func (importer packageImporter) Import(path string) (*types.Package, error) {
	return importer.ImportFrom(path, "", 0)
}

func (importer packageImporter) ImportFrom(path, dir string, _ types.ImportMode) (*types.Package, error) {
	dependency := importer.metadata.Imports[path]
	if dependency != nil && dependency.Module != nil && (dependency.Module.Main || strings.HasPrefix(dependency.ID, dependency.PkgPath+" [")) {
		pkg, err := importer.loader.Check(dependency)
		if err != nil {
			return nil, err
		}
		return pkg.Types, nil
	}
	return gcimporter.Import(importer.loader.fset, importer.loader.imports, path, dir, func(path string) (io.ReadCloser, error) {
		filename := importer.loader.exports[path]
		if filename == "" {
			return nil, fmt.Errorf("no compiled export data for %s", path)
		}
		return os.Open(filename)
	})
}

// CheckGo checks ordinary Go syntax separately from its stock Go type objects.
func CheckGo(metadata *packages.Package, dir string) (*Package, error) {
	loader, err := NewLoader(dir, []*packages.Package{metadata}, nil, nil)
	if err != nil {
		return nil, err
	}
	pkg := &Package{
		Fset:           metadata.Fset,
		Syntax:         metadata.Syntax,
		GoFiles:        metadata.GoFiles,
		TypesInfo:      newInfo(),
		Operations:     make(map[string]types.TGoOperation),
		Comprehensions: make(map[*ast.CallExpr]*frontend.Comprehension),
		Conditionals:   make(map[*ast.CallExpr]ast.Stmt),
	}
	loader.fset = metadata.Fset
	return loader.check(metadata, pkg)
}
