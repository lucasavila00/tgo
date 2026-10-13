package checker

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"os"
	"os/exec"
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
	ShortReturns   map[*ast.ReturnStmt]frontend.ShortReturn
}

// Loader checks source packages and reads compiled dependency types.
// It shares private type objects across those checks and imports.
type Loader struct {
	root     string
	tgoFiles map[string]string
	used     map[string]bool
	packages map[string]*Package
	checking map[string]bool
	imports  map[string]*types.Package
	exports  map[string]string
	fset     *token.FileSet
	sizes    types.Sizes
}

// NewLoader creates one owner for source checks and imported types.
// sizes must describe the module's selected Go target architecture.
func NewLoader(
	root string,
	roots []*packages.Package,
	tgoFiles map[string]string,
	used map[string]bool,
	sizes types.Sizes,
) *Loader {
	return newLoader(root, roots, tgoFiles, used, sizes, token.NewFileSet())
}

// newLoader creates package and import caches and records dependency export files.
// It uses the supplied position set and target sizes for all checks.
func newLoader(root string, roots []*packages.Package, tgoFiles map[string]string,
	used map[string]bool, sizes types.Sizes, fset *token.FileSet) *Loader {
	loader := &Loader{root: root, tgoFiles: tgoFiles, used: used,
		packages: make(map[string]*Package), checking: make(map[string]bool),
		imports: make(map[string]*types.Package), exports: make(map[string]string), fset: fset, sizes: sizes}
	packages.Visit(roots, nil, func(pkg *packages.Package) {
		if pkg.ExportFile != "" {
			loader.exports[pkg.PkgPath] = pkg.ExportFile
		}
	})
	return loader
}

// newInfo allocates the type, binding, scope, and generic-instance maps filled by checking.
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
		ShortReturns:   make(map[*ast.ReturnStmt]frontend.ShortReturn),
	}
	filenames := metadata.CompiledGoFiles
	if len(filenames) == 0 {
		filenames = metadata.GoFiles
	}
	for _, filename := range filenames {
		source, err := loader.parseFile(filename)
		if err != nil {
			return nil, err
		}
		pkg.Syntax = append(pkg.Syntax, source.file)
		if source.added != nil {
			pkg.recordParsedFile(source.added)
		}
	}
	return loader.check(metadata, pkg)
}

// check checks package syntax, records conditional nodes, and caches the successful package.
// Its importer shares type objects with dependencies checked by the same loader.
func (loader *Loader) check(metadata *packages.Package, pkg *Package) (*Package, error) {
	config := types.Config{
		TGo:      pkg.Operations,
		Importer: packageImporter{loader, metadata},
		Sizes:    loader.sizes,
		Error:    func(err error) { pkg.Errors = append(pkg.Errors, err) },
	}
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

// TargetSizes reads the Go target used to check architecture-dependent types.
func TargetSizes(root string) (types.Sizes, error) {
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

// Import resolves an import path through the loader without a source directory.
func (importer packageImporter) Import(path string) (*types.Package, error) {
	return importer.ImportFrom(path, "", 0)
}

// ImportFrom checks source dependencies and reads compiled export data for other imports.
// The loader caches imported packages so their type identities stay consistent.
func (importer packageImporter) ImportFrom(path, dir string, _ types.ImportMode) (*types.Package, error) {
	dependency := importer.metadata.Imports[path]
	if dependency != nil && dependency.Module != nil &&
		(dependency.Module.Main || strings.HasPrefix(dependency.ID, dependency.PkgPath+" [")) {
		pkg, err := importer.loader.Check(dependency)
		if err != nil {
			return nil, err
		}
		return pkg.Types, nil
	}
	return gcimporter.Import(
		importer.loader.fset,
		importer.loader.imports,
		path,
		dir,
		func(path string) (io.ReadCloser, error) {
			filename := importer.loader.exports[path]
			if filename == "" {
				return nil, fmt.Errorf("no compiled export data for %s", path)
			}
			return os.Open(filename)
		},
	)
}

// CheckGo checks ordinary Go syntax separately from its stock Go type objects.
func CheckGo(metadata *packages.Package, dir string) (*Package, error) {
	sizes, err := TargetSizes(dir)
	if err != nil {
		return nil, err
	}
	loader := newLoader(dir, []*packages.Package{metadata}, nil, nil, sizes, metadata.Fset)
	pkg := &Package{
		Fset:           metadata.Fset,
		Syntax:         metadata.Syntax,
		GoFiles:        metadata.GoFiles,
		TypesInfo:      newInfo(),
		Operations:     make(map[string]types.TGoOperation),
		Comprehensions: make(map[*ast.CallExpr]*frontend.Comprehension),
		Conditionals:   make(map[*ast.CallExpr]ast.Stmt),
		ShortReturns:   make(map[*ast.ReturnStmt]frontend.ShortReturn),
	}
	return loader.check(metadata, pkg)
}
