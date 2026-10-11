// Package checker owns source checking with TGo expression rules.
package checker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lucasavila00/tgo/internal/checker/gcimporter"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
	"golang.org/x/tools/go/packages"
)

// Package keeps checked syntax and private type objects together.
type Package struct {
	Fset       *token.FileSet
	Syntax     []*ast.File
	Types      *types.Package
	TypesInfo  *types.Info
	GoFiles    []string
	Errors     []error
	Operations map[string]types.TGoOperation
}

// Loader checks module packages and imports compiled dependencies.
// Every package uses the same private type universe.
type Loader struct {
	Root     string
	TGoFiles map[string]string
	Used     map[string]bool
	packages map[string]*Package
	checking map[string]bool
	imports  map[string]*types.Package
	exports  map[string]string
	fset     *token.FileSet
}

// NewLoader records dependency exports before any source package is checked.
func NewLoader(root string, roots []*packages.Package, tgoFiles map[string]string, used map[string]bool) *Loader {
	loader := &Loader{Root: root, TGoFiles: tgoFiles, Used: used,
		packages: make(map[string]*Package), checking: make(map[string]bool),
		imports: make(map[string]*types.Package), exports: make(map[string]string), fset: token.NewFileSet()}
	packages.Visit(roots, nil, func(pkg *packages.Package) {
		if pkg.ExportFile != "" {
			loader.exports[pkg.PkgPath] = pkg.ExportFile
		}
	})
	return loader
}

func newInfo() *types.Info {
	return &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Implicits: make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection), Scopes: make(map[ast.Node]*types.Scope),
		Instances: make(map[*ast.Ident]types.Instance), FileVersions: make(map[*ast.File]string),
	}
}

// Check checks a selected source package and its module dependencies.
func (loader *Loader) Check(metadata *packages.Package) (*Package, error) {
	if pkg := loader.packages[metadata.ID]; pkg != nil {
		return pkg, nil
	}
	if loader.checking[metadata.ID] {
		return nil, fmt.Errorf("import cycle through %s", metadata.PkgPath)
	}
	loader.checking[metadata.ID] = true
	defer delete(loader.checking, metadata.ID)
	pkg := &Package{Fset: loader.fset, TypesInfo: newInfo(), GoFiles: metadata.GoFiles,
		Operations: make(map[string]types.TGoOperation)}
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
	config := types.Config{TGo: pkg.Operations, Importer: packageImporter{loader, metadata}, Sizes: targetSizes(), Error: func(err error) { pkg.Errors = append(pkg.Errors, err) }}
	if metadata.Module != nil {
		config.GoVersion = "go" + metadata.Module.GoVersion
	}
	typ, err := config.Check(metadata.PkgPath, pkg.Fset, pkg.Syntax, pkg.TypesInfo)
	pkg.Types = typ
	if err != nil {
		return nil, err
	}
	loader.packages[metadata.ID] = pkg
	return pkg, nil
}

func targetSizes() types.Sizes {
	arch := os.Getenv("GOARCH")
	if arch == "" {
		arch = runtime.GOARCH
	}
	return types.SizesFor("gc", arch)
}

type packageImporter struct {
	loader   *Loader
	metadata *packages.Package
}

func (importer packageImporter) Import(path string) (*types.Package, error) {
	return importer.ImportFrom(path, "", 0)
}

func (importer packageImporter) ImportFrom(path, dir string, _ types.ImportMode) (*types.Package, error) {
	dep := importer.metadata.Imports[path]
	if dep != nil && dep.Module != nil && (dep.Module.Main || strings.HasPrefix(dep.ID, dep.PkgPath+" [")) {
		pkg, err := importer.loader.Check(dep)
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
	loader := NewLoader(dir, []*packages.Package{metadata}, nil, nil)
	pkg := &Package{Fset: metadata.Fset, Syntax: metadata.Syntax, GoFiles: metadata.GoFiles,
		TypesInfo: newInfo(), Operations: make(map[string]types.TGoOperation)}
	loader.fset = metadata.Fset
	return loader.check(metadata, pkg)
}
