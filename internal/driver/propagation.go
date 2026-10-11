package driver

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"github.com/lucasavila00/tgo/internal/frontend"
	"github.com/lucasavila00/tgo/internal/lowering"
	"golang.org/x/tools/go/packages"
)

// selectedSources records Go build selection and the selected TGo operations.
type selectedSources struct {
	module      *preparedModule
	activeFiles map[string]bool
	operations  map[string]frontend.Marker
}

// checkedModule carries helper-checked packages and operations to the lowerer.
type checkedModule struct {
	sources    *selectedSources
	packages   []*packages.Package
	operations map[string]lowering.Propagation
}

// selectSources keeps operators only from files selected by Go.
func selectSources(module *preparedModule) *selectedSources {
	initial := loadProject(module.dir)
	activeFiles := make(map[string]bool)
	activeOperations := make(map[string]frontend.Marker)
	for _, pkg := range initial {
		for _, filename := range pkg.GoFiles {
			name, err := filepath.Rel(module.dir, filename)
			must(err)
			if module.tgoFiles[name] {
				activeFiles[name] = true
			}
		}
		for _, file := range pkg.Syntax {
			name, err := filepath.Rel(module.dir, pkg.Fset.PositionFor(file.Pos(), false).Filename)
			must(err)
			if !module.tgoFiles[name] {
				continue
			}
			activeFiles[name] = true
			for _, operation := range frontend.MarkerExpressions(pkg.Fset, file, module.operations) {
				activeOperations[operation.Name] = operation
			}
		}
	}
	return &selectedSources{module: module, activeFiles: activeFiles, operations: activeOperations}
}

// checkPropagations resolves nested calls before the lowerer reads their types.
func checkPropagations(sources *selectedSources) *checkedModule {
	operations := sources.operations
	remaining := len(operations)
	for remaining > 0 {
		pkgs := loadProject(sources.module.dir)
		progress := false
		written := make(map[string]bool)
		for _, pkg := range pkgs {
			for _, file := range pkg.Syntax {
				filename := pkg.Fset.PositionFor(file.Pos(), false).Filename
				if written[filename] {
					continue
				}
				selected, operation := nextPropagation(pkg, file, operations)
				if selected == nil {
					continue
				}
				count := successResultCount(pkg.TypesInfo.TypeOf(ast.Unparen(selected)))
				frontend.WrapMarker(file, selected, operation)
				file.Decls = append(file.Decls, frontend.MarkerDeclaration(operation, count))
				writeSyntax(pkg.Fset, file)
				written[filename] = true
				operations[operation.Name] = operation
				remaining--
				progress = true
			}
		}
		if !progress {
			panic("cannot type check propagation source")
		}
	}
	loweredOperations := make(map[string]lowering.Propagation, len(operations))
	for name, operation := range sources.operations {
		loweredOperations[name] = lowering.Propagation{Wrap: operation.Wrap, CallName: operation.CallName, Position: operation.Position}
	}
	return &checkedModule{sources: sources, packages: loadProject(sources.module.dir), operations: loweredOperations}
}

// nextPropagation selects the smallest typed call, then the earliest one.
func nextPropagation(pkg *packages.Package, file *ast.File, operations map[string]frontend.Marker) (ast.Expr, frontend.Marker) {
	candidates := frontend.MarkerExpressions(pkg.Fset, file, operations)
	var selected ast.Expr
	var operation frontend.Marker
	for expression, op := range candidates {
		operations[op.Name] = op
		typ := pkg.TypesInfo.TypeOf(ast.Unparen(expression))
		if typ == nil || typ == types.Typ[types.Invalid] {
			continue
		}
		if selected == nil || expression.End()-expression.Pos() < selected.End()-selected.Pos() || expression.End()-expression.Pos() == selected.End()-selected.Pos() && expression.Pos() < selected.Pos() {
			selected = expression
			operation = op
		}
	}
	return selected, operation
}

// successResultCount checks the final error result and counts the other results.
func successResultCount(result types.Type) int {
	count := 1
	last := result
	if tuple, ok := result.(*types.Tuple); ok {
		count = tuple.Len()
		if count == 0 {
			panic("propagation requires an error result")
		}
		last = tuple.At(count - 1).Type()
	}
	if !types.AssignableTo(last, types.Universe.Lookup("error").Type()) {
		panic("propagation requires a final error result")
	}
	return count - 1
}

// lowerModule checks return targets and writes lowered package files.
func lowerModule(module *checkedModule) {
	for _, pkg := range module.packages {
		changed := false
		for _, filename := range pkg.GoFiles {
			name, err := filepath.Rel(module.sources.module.dir, filename)
			must(err)
			changed = changed || module.sources.activeFiles[name]
		}
		if !changed {
			continue
		}
		rejectUnsafePropagation(pkg, module.sources.operations)
		checkPropagationReturns(pkg, module.sources.operations)
		source, err := lowering.SourceFromPackage(pkg, module.sources.module.dir)
		must(err)
		output, err := source.LowerPropagations(module.operations)
		must(err)
		for name, content := range output {
			must(os.WriteFile(name, content, 0600))
		}
	}
}

func writeSyntax(fset *token.FileSet, file *ast.File) {
	var buf bytes.Buffer
	must(format.Node(&buf, fset, file))
	must(os.WriteFile(fset.PositionFor(file.Pos(), false).Filename, buf.Bytes(), 0600))
}
