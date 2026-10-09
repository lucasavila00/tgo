package compiler

import (
	"go/ast"
	"go/token"
	"go/types"

	"tgo/pkg/syntax"
)

// AnalysisSource pairs source syntax with its typed Go projection.
type AnalysisSource struct {
	Syntax    *syntax.File
	Projected *ast.File
	Generated map[ast.Decl]bool
}

// AnalysisPackage contains checked TGo source and its typed projection.
type AnalysisPackage struct {
	Sources []AnalysisSource
	FileSet *token.FileSet
	Info    *types.Info
	Package *types.Package
	NonNil  map[token.Pos]bool
}

// AnalyzePackage loads and checks one TGo package without writing output files.
func AnalyzePackage(
	directory string,
	importPath string,
	files *token.FileSet,
) (*AnalysisPackage, error) {
	root, module, err := moduleRoot(directory)
	if err != nil {
		return nil, err
	}
	context, err := effectiveBuildContext(directory)
	if err != nil {
		return nil, err
	}
	packages, err := discover(root, module, &context)
	if err != nil {
		return nil, err
	}
	unit := packages[importPath]
	if unit == nil {
		return nil, nil
	}
	if files != nil {
		unit.fs = files
	}
	if err := loadAnalysisPackage(unit, packages, make(map[string]bool)); err != nil {
		return nil, err
	}
	if len(unit.Sources) == 0 {
		return nil, nil
	}
	if err := unit.checkAndLower(); err != nil {
		return nil, err
	}
	sources := make([]AnalysisSource, 0, len(unit.Sources))
	nonNil := make(map[token.Pos]bool)
	for _, source := range unit.Sources {
		generated := make(map[ast.Decl]bool)
		for _, declaration := range source.File.Decls {
			if unit.generatedDecl(declaration) {
				generated[declaration] = true
			}
		}
		sources = append(sources, AnalysisSource{
			Syntax: source.Tree, Projected: source.File, Generated: generated,
		})
		for position := range source.NonNil {
			nonNil[position] = true
		}
	}
	return &AnalysisPackage{
		Sources: sources, FileSet: unit.fs, Info: unit.info,
		Package: unit.typed, NonNil: nonNil,
	}, nil
}

func loadAnalysisPackage(
	unit *packageUnit,
	packages map[string]*packageUnit,
	loaded map[string]bool,
) error {
	if loaded[unit.Path] {
		return nil
	}
	loaded[unit.Path] = true
	if err := unit.load(); err != nil {
		return err
	}
	for _, path := range importsOf(unit.Files) {
		dependency := packages[path]
		if dependency != nil {
			if err := loadAnalysisPackage(dependency, packages, loaded); err != nil {
				return err
			}
		}
	}
	return nil
}
