package compiler

import (
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"

	"tgo/internal/sourcefacts"
	"tgo/pkg/syntax"
)

// AnalysisSource contains source syntax and its generated output.
type AnalysisSource struct {
	Name   string
	Output []byte
	Syntax *syntax.File
}

// AnalysisPackage contains checked TGo source and indexed type facts.
type AnalysisPackage struct {
	Sources []AnalysisSource
	Facts   *sourcefacts.Index
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
	outputs, err := unit.generatedOutputs()
	if err != nil {
		return nil, err
	}
	if unit.info == nil || unit.fs == nil {
		return nil, fmt.Errorf("analysis package has no type or position facts")
	}
	sources := make([]AnalysisSource, 0, len(unit.Sources))
	nonNil := make(map[token.Pos]bool)
	var facts *sourcefacts.Index
	for _, source := range unit.Sources {
		tree := source.Tree
		if tree == nil {
			return nil, fmt.Errorf("analysis source %s has no syntax", source.Name)
		}
		sources = append(sources, AnalysisSource{
			Name: filepath.Base(source.Name), Output: outputs[unit.outputPath(source.Name)],
			Syntax: tree,
		})
		if facts == nil {
			facts = sourcefacts.New(tree, unit.info, unit.fs)
		} else {
			facts.AddFile(tree)
		}
		for position := range source.NonNil {
			nonNil[position] = true
		}
	}
	return &AnalysisPackage{
		Sources: sources, Facts: facts,
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
