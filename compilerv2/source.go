package compilerv2

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// Site identifies an expression in the original source. Offsets are bytes.
type Site struct {
	File        string `json:"file"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Kind        string `json:"kind"`
	Reason      string `json:"reason,omitempty"`
	ErrorReturn bool   `json:"error_return"`
}

// Source contains the original syntax and its type information.
type Source struct {
	Package     *packages.Package
	Sites       []Site
	expressions map[Site]ast.Expr
}

// Load reads one Go package. It does not change its syntax tree.
func Load(dir, pattern string) (*Source, error) {
	pkgs, err := packages.Load(&packages.Config{
		Dir: dir,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedImports | packages.NeedDeps,
	}, pattern)
	if err != nil {
		return nil, err
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected one package, got %d", len(pkgs))
	}
	p := pkgs[0]
	if len(p.Errors) != 0 {
		return nil, fmt.Errorf("load source: %s", p.Errors[0])
	}
	s := &Source{Package: p, expressions: make(map[Site]ast.Expr)}
	for _, file := range p.Syntax {
		s.inventory(file, dir)
	}
	return s, nil
}

func (s *Source) inventory(file *ast.File, dir string) {
	var stack []ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if expr, ok := n.(ast.Expr); ok {
			start := s.Package.Fset.Position(expr.Pos())
			end := s.Package.Fset.Position(expr.End())
			name, err := filepath.Rel(dir, start.Filename)
			if err != nil {
				name = start.Filename
			}
			site := Site{File: name, Start: start.Offset, End: end.Offset, Kind: fmt.Sprintf("%T", expr)}
			tv, hasType := s.Package.TypesInfo.Types[expr]
			switch {
			case hasType && tv.IsType():
				site.Reason = "type syntax"
			case !hasType:
				site.Reason = "name or syntax without a value"
			}
			var fn ast.Node
			for i := len(stack) - 1; i >= 0; i-- {
				switch parent := stack[i].(type) {
				case *ast.FuncDecl, *ast.FuncLit:
					fn = parent
				case *ast.GenDecl:
					if parent.Tok == token.CONST {
						site.Reason = "constant declaration"
					}
				case *ast.ArrayType:
					if contains(parent.Len, expr) {
						site.Reason = "array length"
					}
				case *ast.GoStmt:
					if parent.Call == expr {
						site.Reason = "call executes in another goroutine"
					}
				case *ast.DeferStmt:
					if parent.Call == expr {
						site.Reason = "call executes when the function returns"
					}
				}
				if fn != nil {
					break
				}
			}
			if fn == nil && site.Reason == "" {
				site.Reason = "package scope"
			}
			if fn != nil {
				var typ *ast.FuncType
				switch f := fn.(type) {
				case *ast.FuncDecl:
					typ = f.Type
				case *ast.FuncLit:
					typ = f.Type
				}
				sig, _ := s.Package.TypesInfo.TypeOf(typ).(*types.Signature)
				if sig == nil {
					if f, ok := fn.(*ast.FuncDecl); ok {
						sig, _ = s.Package.TypesInfo.Defs[f.Name].Type().(*types.Signature)
					}
					if f, ok := fn.(*ast.FuncLit); ok {
						sig, _ = s.Package.TypesInfo.TypeOf(f).(*types.Signature)
					}
				}
				if sig != nil && sig.Results().Len() == 1 {
					site.ErrorReturn = types.Identical(sig.Results().At(0).Type(), types.Universe.Lookup("error").Type())
				}
			}
			s.Sites = append(s.Sites, site)
			s.expressions[site] = expr
		}
		stack = append(stack, n)
		return true
	})
}

func contains(parent, child ast.Node) bool {
	return parent != nil && parent.Pos() <= child.Pos() && child.End() <= parent.End()
}
