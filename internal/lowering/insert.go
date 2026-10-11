package lowering

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
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
	output, err := r.emit()
	if err != nil {
		return nil, err
	}
	if !r.inserted {
		return nil, fmt.Errorf("expression site was not lowered: %s:%d", site.File, site.Start)
	}
	return output, nil
}

// LowerPropagations emits source from typed marker calls.
func (s *Source) LowerPropagations(operations map[string]Propagation) (map[string][]byte, error) {
	r := &rewrite{source: s, names: make(map[string]bool), propagations: make(map[ast.Expr]Propagation)}
	for site, expression := range s.expressions {
		call, ok := expression.(*ast.CallExpr)
		if !ok {
			continue
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok {
			continue
		}
		operation, ok := operations[id.Name]
		if !ok {
			continue
		}
		if site.Reason != "" {
			return nil, fmt.Errorf("propagation has no current-function evaluation: %s", site.Reason)
		}
		r.propagations[call] = operation
	}
	output, err := r.emit()
	if err != nil {
		return nil, err
	}
	// Keep only marker declarations that the type checker still needs.
	used := make(map[string]bool)
	parsed := make(map[string]*ast.File)
	fset := token.NewFileSet()
	for name, content := range output {
		file, err := parser.ParseFile(fset, name, content, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		parsed[name] = file
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok {
					used[id.Name] = true
				}
			}
			return true
		})
	}
	for name, file := range parsed {
		declarations := file.Decls[:0]
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if _, marker := operations[fn.Name.Name]; marker && !used[fn.Name.Name] {
					continue
				}
			}
			declarations = append(declarations, decl)
		}
		file.Decls = declarations
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, file); err != nil {
			return nil, err
		}
		output[name] = buf.Bytes()
	}
	return output, nil
}

func (r *rewrite) contains(node ast.Node) bool {
	if r.suppressed {
		return false
	}
	if contains(node, r.target) {
		return true
	}
	for expression := range r.propagations {
		if contains(node, expression) {
			return true
		}
	}
	return false
}

func (r *rewrite) isTarget(expression ast.Expr) bool {
	if r.suppressed {
		return false
	}
	if expression == r.target && expression != nil {
		return true
	}
	_, ok := r.propagations[expression]
	return ok
}

func (r *rewrite) emit() (map[string][]byte, error) {
	for _, file := range r.source.Package.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok {
				r.names[id.Name] = true
			}
			return true
		})
	}
	output := make(map[string][]byte)
	for _, file := range r.source.Package.Syntax {
		r.extraImports = nil
		r.extraDeclarations = nil
		r.formatAlias = nil
		r.zeroSequence = nil
		r.zeroPointer = nil
		r.iterAlias = nil
		r.mapAlias = nil
		r.sliceAlias = nil
		r.reflectAlias = nil
		r.typeImports = nil
		copy := *file
		copy.Decls = append([]ast.Decl(nil), file.Decls...)
		for i, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && r.contains(fn.Body) {
				function := *fn
				function.Type, function.Body = r.function(fn, fn.Type, fn.Body)
				copy.Decls[i] = &function
			}
		}
		if r.err != nil {
			return nil, r.err
		}
		copy.Decls = append(copy.Decls, r.extraDeclarations...)
		for _, spec := range r.extraImports {
			copy.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}}, copy.Decls...)
		}
		var buf bytes.Buffer
		if err := format.Node(&buf, r.source.Package.Fset, &copy); err != nil {
			return nil, err
		}
		output[r.source.Package.Fset.Position(file.Pos()).Filename] = buf.Bytes()
	}
	return output, nil
}
