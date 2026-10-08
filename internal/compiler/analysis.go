package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

// AnalysisPackage contains checked TGo source before Go output is formatted.
type AnalysisPackage struct {
	Files   []*ast.File
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
	sourceFiles := make([]*ast.File, 0, len(unit.Sources))
	nonNil := make(map[token.Pos]bool)
	for _, source := range unit.Sources {
		sourceFiles = append(sourceFiles, source.File)
		for position := range source.NonNil {
			nonNil[position] = true
		}
	}
	return &AnalysisPackage{
		Files: sourceFiles, FileSet: unit.fs, Info: unit.info,
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
