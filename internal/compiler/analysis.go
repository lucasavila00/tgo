package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"path/filepath"
	"sort"

	"tgo/internal/sourcefacts"
	"tgo/pkg/syntax"
)

// AnalysisSource contains source syntax and its generated output.
type AnalysisSource struct {
	Name   string
	Path   string
	Output []byte
	Syntax *syntax.File
}

// AnalysisPackage contains checked TGo source and indexed type facts.
type AnalysisPackage struct {
	Directory string
	Path      string
	Sources   []AnalysisSource
	Facts     *sourcefacts.Index
	Files     *token.FileSet
	Package   *types.Package
	NonNil    map[token.Pos]bool
}

// AnalyzeWorkspace loads and checks all active TGo packages in one module.
func AnalyzeWorkspace(directory string) ([]*AnalysisPackage, error) {
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
	paths := make([]string, 0, len(packages))
	for path := range packages {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	loaded := make(map[string]bool)
	result := make([]*AnalysisPackage, 0, len(paths))
	for _, path := range paths {
		unit := packages[path]
		if err := loadAnalysisPackage(unit, packages, loaded); err != nil {
			return nil, err
		}
		if len(unit.Sources) == 0 {
			continue
		}
		analysis, err := analyzeUnit(unit)
		if err != nil {
			return nil, err
		}
		result = append(result, analysis)
	}
	return result, nil
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
	return analyzeUnit(unit)
}

func analyzeUnit(unit *packageUnit) (*AnalysisPackage, error) {
	if err := unit.checkAndLower(); err != nil {
		return nil, err
	}
	outputs, err := unit.generatedOutputs()
	if err != nil {
		return nil, err
	}
	sources, facts, nonNil, err := analysisSources(unit, outputs)
	if err != nil {
		return nil, err
	}
	return &AnalysisPackage{
		Directory: unit.Dir,
		Path:      unit.Path,
		Sources:   sources,
		Facts:     facts,
		Files:     unit.fs,
		Package:   unit.typed,
		NonNil:    nonNil,
	}, nil
}

func analysisSources(
	unit *packageUnit,
	outputs map[string][]byte,
) ([]AnalysisSource, *sourcefacts.Index, map[token.Pos]bool, error) {
	if unit.info == nil || unit.fs == nil {
		return nil, nil, nil, fmt.Errorf("analysis package has no type or position facts")
	}
	sources := make([]AnalysisSource, 0, len(unit.Sources))
	nonNil := make(map[token.Pos]bool)
	info := analysisTypeInfo(unit)
	var facts *sourcefacts.Index
	for _, source := range unit.Sources {
		tree := source.Tree
		if tree == nil {
			return nil, nil, nil, fmt.Errorf("analysis source %s has no syntax", source.Name)
		}
		sources = append(sources, AnalysisSource{
			Name: filepath.Base(source.Name), Path: source.Name,
			Output: outputs[unit.outputPath(source.Name)],
			Syntax: tree,
		})
		if facts == nil {
			facts = sourcefacts.New(tree, info, unit.fs)
		} else {
			facts.AddFile(tree)
		}
		for position := range source.NonNil {
			nonNil[position] = true
		}
	}
	return sources, facts, nonNil, nil
}

// analysisTypeInfo removes generated function facts that can share source positions.
func analysisTypeInfo(unit *packageUnit) *types.Info {
	result := *unit.info
	result.Types = maps.Clone(unit.info.Types)
	result.Defs = maps.Clone(unit.info.Defs)
	result.Uses = maps.Clone(unit.info.Uses)
	result.Implicits = maps.Clone(unit.info.Implicits)
	result.Selections = maps.Clone(unit.info.Selections)
	result.Scopes = maps.Clone(unit.info.Scopes)
	result.Instances = maps.Clone(unit.info.Instances)
	for declaration := range unit.generated {
		if _, ok := declaration.(*ast.FuncDecl); !ok {
			continue
		}
		ast.Inspect(declaration, func(node ast.Node) bool {
			delete(result.Implicits, node)
			delete(result.Scopes, node)
			if expression, ok := node.(ast.Expr); ok {
				delete(result.Types, expression)
			}
			if identifier, ok := node.(*ast.Ident); ok {
				delete(result.Defs, identifier)
				delete(result.Uses, identifier)
				delete(result.Instances, identifier)
			}
			if selector, ok := node.(*ast.SelectorExpr); ok {
				delete(result.Selections, selector)
			}
			return true
		})
	}
	return &result
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
