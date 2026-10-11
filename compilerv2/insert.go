package compilerv2

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
)

// InsertAfter builds source files with statements after one expression.
// The callback uses the expression's evaluated results in source order.
func (s *Source) InsertAfter(site Site, after After) (map[string][]byte, error) {
	return s.insert(site, after, nil)
}

func (s *Source) insert(site Site, after After, propagation *propagation) (map[string][]byte, error) {
	target, ok := s.expressions[site]
	if !ok {
		return nil, fmt.Errorf("expression site does not belong to this source")
	}
	if site.Reason != "" {
		return nil, fmt.Errorf("expression has no current-function evaluation: %s", site.Reason)
	}
	r := &rewrite{source: s, target: target, after: after, names: make(map[string]bool), propagation: propagation}
	for _, file := range s.Package.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				r.names[id.Name] = true
			}
			return true
		})
	}
	if propagation != nil {
		for node := s.parents[target]; node != nil; node = s.parents[node] {
			switch node.(type) {
			case *ast.FuncDecl, *ast.FuncLit:
				r.returnFunction = node
			}
			if r.returnFunction != nil {
				break
			}
		}
		if r.returnFunction == nil {
			return nil, fmt.Errorf("propagation requires a function")
		}
		if propagation.wrap {
			r.formatAlias = r.name()
		}
	}
	output := make(map[string][]byte)
	for _, file := range s.Package.Syntax {
		copy := *file
		copy.Decls = append([]ast.Decl(nil), file.Decls...)
		for i, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && contains(fn.Body, target) {
				function := *fn
				var prefix []ast.Stmt
				if fn == r.returnFunction {
					function.Type, prefix = r.prepareReturns(fn.Type)
				}
				function.Body = r.block(fn.Body)
				function.Body.List = append(prefix, function.Body.List...)
				copy.Decls[i] = &function
			}
		}
		if propagation != nil {
			if r.formatAlias != nil && contains(file, target) {
				spec := &ast.ImportSpec{Name: r.formatAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: "\"fmt\""}}
				copy.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}}, copy.Decls...)
			}
		}
		var buf bytes.Buffer
		if err := format.Node(&buf, s.Package.Fset, &copy); err != nil {
			return nil, err
		}
		name := s.Package.Fset.Position(file.Pos()).Filename
		output[name] = buf.Bytes()
	}
	if !r.inserted {
		return nil, fmt.Errorf("expression site was not lowered: %s:%d", site.File, site.Start)
	}
	return output, nil
}
