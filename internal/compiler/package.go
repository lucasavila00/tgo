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
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type packageUnit struct {
	Dir, Path       string
	Sources         []*source
	Files           []*ast.File
	Models          map[string]*model
	Imports         map[string]*packageUnit
	sourcePaths     []string
	matchingPaths   []string
	generatedPaths  []string
	knownOS         map[string]bool
	knownArch       map[string]bool
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
	erasedImports   map[*ast.ImportSpec]bool
	references      []generatedReference
	usedIdentifiers map[string]bool
	typeErrors      []error
	errors          []error
	serial          int
}

// outputPath returns the Go output path while preserving target suffixes.
func (p *packageUnit) outputPath(sourcePath string) string {
	directory := filepath.Dir(sourcePath)
	name := strings.TrimSuffix(filepath.Base(sourcePath), ".tgo")
	parts := strings.Split(name, "_")
	suffix := len(parts)
	if len(parts) > 2 && p.knownOS[parts[len(parts)-2]] &&
		p.knownArch[parts[len(parts)-1]] {
		suffix = len(parts) - 2
	} else if len(parts) > 1 &&
		(p.knownOS[parts[len(parts)-1]] || p.knownArch[parts[len(parts)-1]]) {
		suffix = len(parts) - 1
	}
	if suffix == len(parts) {
		return filepath.Join(directory, name+"_tgo.go")
	}
	prefix := strings.Join(parts[:suffix], "_")
	target := strings.Join(parts[suffix:], "_")
	return filepath.Join(directory, prefix+"_tgo_"+target+".go")
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
	cache := map[string]string{}
	imp := importer.ForCompiler(p.fs, "gc", func(path string) (io.ReadCloser, error) {
		export, ok := cache[path]
		if !ok {
			cmd := exec.Command("go", "list", "-export", "-f", "{{.Export}}", path)
			cmd.Dir = p.Dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("load %s: %s", path, out)
			}
			export = strings.TrimSpace(string(out))
			cache[path] = export
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

// compile lowers one tgo package and formats its Go output files.
func (p *packageUnit) compile() (map[string][]byte, error) {
	p.prepare()
	if len(p.errors) > 0 {
		return nil, p.errors[0]
	}
	p.typecheck()
	p.checkGeneratedPredeclaredNames()
	if len(p.errors) > 0 {
		return nil, p.errors[0]
	}
	p.lowerConstructions()
	p.typecheck()
	if p.blankUnusedErasedImports() {
		p.typecheck()
	}
	p.validateGeneratedReferences()
	if len(p.errors) > 0 {
		return nil, p.errors[0]
	}
	p.fillDefaults()
	p.lowerMatches()
	if len(p.errors) > 0 {
		return nil, p.errors[0]
	}
	p.typecheck()
	p.validateGeneratedReferences()
	if len(p.errors) > 0 {
		return nil, p.errors[0]
	}
	if len(p.typeErrors) > 0 {
		return nil, p.typeErrors[0]
	}
	p.checkRules()
	if len(p.errors) > 0 {
		return nil, p.errors[0]
	}
	outputs := map[string][]byte{}
	for _, s := range p.Sources {
		removeLineDirectives(s.File)
		var b bytes.Buffer
		b.WriteString(generatedHeader + "\n\n")
		if err := format.Node(&b, p.fs, s.File); err != nil {
			return nil, err
		}
		outputs[p.outputPath(s.Name)] = b.Bytes()
	}
	return outputs, nil
}
