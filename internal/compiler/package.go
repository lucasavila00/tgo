package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type packageUnit struct {
	Dir, Path      string
	Sources        []*source
	Files          []*ast.File
	Models         map[string]*model
	Imports        map[string]*packageUnit
	sourcePaths    []string
	matchingPaths  []string
	generatedPaths []string
	knownOS        map[string]bool
	knownArch      map[string]bool
	sourcesMatched bool
	loaded         bool
	matchError     error
	loadError      error
	fs             *token.FileSet
	info           *types.Info
	typed          *types.Package
	generated      map[ast.Decl]bool
	errors         []error
	serial         int
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
	message := fmt.Sprintf(pattern, args...)
	p.errors = append(p.errors, fmt.Errorf("%s: %s", p.fs.Position(n.Pos()), message))
}

// newInfo makes the maps that the Go type checker fills.
func newInfo() *types.Info {
	return &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
}

// typecheck checks all Go and lowered tgo files in one package.
func (p *packageUnit) typecheck(strict bool) error {
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
	conf := types.Config{Importer: imp, Error: func(e error) { problems = append(problems, e) }}
	p.typed, _ = conf.Check(p.Path, p.fs, p.Files, p.info)
	if strict && len(problems) > 0 {
		return problems[0]
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

// modelForType returns tgo metadata for a local or imported named type.
func (p *packageUnit) modelForType(t types.Type) *model {
	t = types.Unalias(t)
	n, ok := t.(*types.Named)
	if !ok {
		return nil
	}
	owner := p
	if n.Obj().Pkg() != nil && n.Obj().Pkg().Path() != p.Path {
		owner = p.Imports[n.Obj().Pkg().Path()]
	}
	if owner == nil {
		return nil
	}
	return owner.Models[n.Obj().Name()]
}

// modelExpr resolves a type expression to its tgo package and model.
func (p *packageUnit) modelExpr(f *ast.File, e ast.Expr) (*packageUnit, *model, string) {
	switch x := e.(type) {
	case *ast.Ident:
		return p, p.Models[x.Name], ""
	case *ast.SelectorExpr:
		id, ok := x.X.(*ast.Ident)
		if !ok {
			return nil, nil, ""
		}
		for _, im := range f.Imports {
			path, _ := strconv.Unquote(im.Path.Value)
			dep := p.Imports[path]
			if dep == nil {
				continue
			}
			alias := filepath.Base(path)
			if len(dep.Sources) > 0 {
				alias = dep.Sources[0].File.Name.Name
			}
			if im.Name != nil {
				alias = im.Name.Name
			}
			if alias == id.Name {
				return dep, dep.Models[x.Sel.Name], alias
			}
		}
	}
	return nil, nil, ""
}

// qualify makes a local name or a package-qualified name.
func qualify(prefix, name string) ast.Expr {
	if prefix == "" {
		return ast.NewIdent(name)
	}
	return &ast.SelectorExpr{X: ast.NewIdent(prefix), Sel: ast.NewIdent(name)}
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
	if err := p.typecheck(false); err != nil {
		return nil, err
	}
	p.fillDefaults()
	p.lowerMatches()
	if len(p.errors) > 0 {
		return nil, p.errors[0]
	}
	if err := p.typecheck(true); err != nil {
		return nil, err
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
