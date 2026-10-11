package compilerv2

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
)

// InsertAfter builds source files with statements after one expression.
// The callback uses the expression's evaluated results in source order.
func (s *Source) InsertAfter(site Site, after After) (map[string][]byte, error) {
	target, ok := s.expressions[site]
	if !ok {
		return nil, fmt.Errorf("expression site does not belong to this source")
	}
	if site.Reason != "" {
		return nil, fmt.Errorf("expression has no current-function evaluation: %s", site.Reason)
	}
	r := &rewrite{source: s, target: target, after: after, names: make(map[string]bool)}
	for _, file := range s.Package.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				r.names[id.Name] = true
			}
			return true
		})
	}
	output := make(map[string][]byte)
	for _, file := range s.Package.Syntax {
		copy := *file
		copy.Decls = append([]ast.Decl(nil), file.Decls...)
		for i, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && contains(fn.Body, target) {
				function := *fn
				function.Body = r.block(fn.Body)
				copy.Decls[i] = &function
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
