package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"strings"

	"tgo/internal/outputname"

	"golang.org/x/tools/go/ast/astutil"
)

type packageUnit struct {
	Dir, Path       string
	Module          string
	Sources         []*source
	Files           []*ast.File
	Models          map[string]*model
	Imports         map[string]*packageUnit
	sourcePaths     []string
	matchingPaths   []string
	generatedPaths  []string
	context         *build.Context
	usesC           bool
	sourcesMatched  bool
	loaded          bool
	matchError      error
	loadError       error
	fs              *token.FileSet
	info            *types.Info
	typed           *types.Package
	generated       map[ast.Decl]bool
	generatedValues map[*ast.ValueSpec]bool
	checkedLiterals map[*ast.CompositeLit]bool
	erasedImports   map[*ast.ImportSpec]bool
	references      []generatedReference
	usedIdentifiers map[string]bool
	exportPaths     map[string]string
	typeErrors      []error
	errors          []error
}

// outputPath returns the Go output path while preserving target suffixes.
func (p *packageUnit) outputPath(sourcePath string) string {
	return outputname.Path(sourcePath)
}

// fail records a source error for later reporting.
func (p *packageUnit) fail(n ast.Node, pattern string, args ...any) {
	p.failAt(n.Pos(), pattern, args...)
}

// failAt records a source error at one stable token position.
func (p *packageUnit) failAt(position token.Pos, pattern string, args ...any) {
	message := fmt.Sprintf(pattern, args...)
	p.errors = append(p.errors, fmt.Errorf("%s: %s", p.fs.Position(position), message))
}

// newInfo makes the maps that the Go type checker fills.
func newInfo() *types.Info {
	return &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
}

// typecheck checks all Go and lowered tgo files in one package.
func (p *packageUnit) typecheck() {
	p.info = newInfo()
	var problems []error
	exportError := p.loadExportPaths()
	imp := importer.ForCompiler(p.fs, "gc", func(path string) (io.ReadCloser, error) {
		if exportError != nil {
			return nil, exportError
		}
		export := p.exportPaths[path]
		if export == "" {
			return nil, fmt.Errorf("load %s: missing export data", path)
		}
		return os.Open(export)
	})
	conf := types.Config{
		Importer:    imp,
		FakeImportC: p.usesC,
		Error:       func(e error) { problems = append(problems, e) },
	}
	p.typed, _ = conf.Check(p.Path, p.fs, p.Files, p.info)
	p.typeErrors = problems
}

// loadExportPaths finds import files with one Go command.
func (p *packageUnit) loadExportPaths() error {
	if p.exportPaths == nil {
		p.exportPaths = make(map[string]string)
	}
	missing := make([]string, 0)
	for _, path := range importsOf(p.Files) {
		if p.exportPaths[path] == "" {
			missing = append(missing, path)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	arguments := make([]string, 0, 5+len(missing))
	arguments = append(
		arguments,
		"list", "-deps", "-export", "-f",
		"{{if .Export}}{{.ImportPath}}\t{{.Export}}{{end}}",
	)
	command := exec.Command("go", append(arguments, missing...)...)
	command.Dir = p.Dir
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("load imports: %s", output)
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		path, export, found := strings.Cut(line, "\t")
		if found {
			p.exportPaths[path] = export
		}
	}
	return nil
}

// transform replaces nodes after visiting their children.
func transform(node ast.Node, change func(ast.Node) ast.Node) ast.Node {
	return astutil.Apply(node, nil, func(cursor *astutil.Cursor) bool {
		if cursor.Node() != nil {
			if replacement := change(cursor.Node()); replacement != cursor.Node() {
				cursor.Replace(replacement)
			}
		}
		return true
	})
}

// modelOwner returns the package and metadata for one tgo named type.
func (p *packageUnit) modelOwner(t types.Type) (*packageUnit, *model) {
	t = types.Unalias(t)
	n, ok := t.(*types.Named)
	if !ok {
		return nil, nil
	}
	owner := p
	if n.Obj().Pkg() != nil && n.Obj().Pkg().Path() != p.Path {
		owner = p.Imports[n.Obj().Pkg().Path()]
	}
	if owner == nil {
		return nil, nil
	}
	return owner, owner.Models[n.Obj().Name()]
}

// modelForType returns tgo metadata for a local or imported named type.
func (p *packageUnit) modelForType(t types.Type) *model {
	_, declaration := p.modelOwner(t)
	return declaration
}

// call makes a Go call expression.
func call(e ast.Expr, args ...ast.Expr) *ast.CallExpr { return &ast.CallExpr{Fun: e, Args: args} }

// generatedDecl reports whether tgo created a declaration.
func (p *packageUnit) generatedDecl(d ast.Decl) bool { return p.generated[d] }

// compile lowers one tgo package and emits its Go output files.
func (p *packageUnit) compile() (map[string][]byte, error) {
	if err := p.checkAndLower(); err != nil {
		return nil, err
	}
	return p.generatedOutputs()
}

// generatedOutputs emits each checked source to its reserved output name.
func (p *packageUnit) generatedOutputs() (map[string][]byte, error) {
	outputs := map[string][]byte{}
	for _, s := range p.Sources {
		if !s.Lowered {
			outputs[p.outputPath(s.Name)] = append([]byte(nil), s.Data...)
			continue
		}
		var body bytes.Buffer
		removeLineDirectives(s.File)
		if err := format.Node(&body, p.fs, s.File); err != nil {
			return nil, err
		}
		var b bytes.Buffer
		b.WriteString(generatedHeader + "\n\n")
		b.Write(body.Bytes())
		outputs[p.outputPath(s.Name)] = b.Bytes()
	}
	return outputs, nil
}

// checkAndLower checks one package and applies all source transformations.
func (p *packageUnit) checkAndLower() error {
	p.prepare()
	p.typecheck()
	p.checkGeneratedPredeclaredNames()
	p.checkCheckedStructs()
	if len(p.errors) > 0 {
		return p.errors[0]
	}
	p.lowerConstructions()
	p.typecheck()
	p.lowerPropagations()
	if len(p.errors) > 0 {
		return p.errors[0]
	}
	p.typecheck()
	if p.blankUnusedErasedImports() {
		p.typecheck()
	}
	p.validateGeneratedReferences()
	if len(p.errors) > 0 {
		return p.errors[0]
	}
	p.fillDefaults()
	p.typecheck()
	p.validateGeneratedReferences()
	if len(p.errors) > 0 {
		return p.errors[0]
	}
	if len(p.typeErrors) > 0 {
		return p.typeErrors[0]
	}
	p.checkEnumJSONFields()
	if len(p.errors) > 0 {
		return p.errors[0]
	}
	p.applyEnumLayouts()
	p.typecheck()
	if len(p.typeErrors) > 0 {
		return p.typeErrors[0]
	}
	p.checkRules()
	if len(p.errors) > 0 {
		return p.errors[0]
	}
	return nil
}
