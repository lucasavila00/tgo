package lowering

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"reflect"
)

// InsertAfter returns Go files with the callback's statements after the selected expression.
// The callback receives saved results; the original syntax tree stays unchanged.
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

// LowerPropagations replaces checked propagation and if values with ordinary Go.
// It returns formatted file contents by absolute path without writing files.
// It reports an error if any accepted operation was not replaced.
func (s *Source) LowerPropagations(operations map[string]Propagation) (map[string][]byte, error) {
	r := &rewrite{source: s, names: make(map[string]bool), propagations: make(map[ast.Expr]Propagation), conditionals: s.checked.Conditionals, lowered: make(map[ast.Expr]bool)}
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
	for _, site := range s.Sites {
		expression := s.expressions[site]
		if _, expected := r.propagations[expression]; expected && !r.lowered[expression] {
			return nil, fmt.Errorf("%s: propagation was not lowered", s.checked.Fset.Position(expression.Pos()))
		}
	}
	for call := range r.conditionals {
		if !r.lowered[call] {
			return nil, fmt.Errorf("%s: if expression was not lowered", s.checked.Fset.Position(call.Pos()))
		}
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
	for expression := range r.conditionals {
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
	call, _ := expression.(*ast.CallExpr)
	return ok || r.conditionals[call] != nil
}

func (r *rewrite) emit() (map[string][]byte, error) {
	for _, file := range r.source.checked.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok {
				r.names[id.Name] = true
			}
			return true
		})
	}
	output := make(map[string][]byte)
	for _, file := range r.source.checked.Syntax {
		r.extraImports = nil
		r.extraDeclarations = nil
		r.formatAlias = nil
		r.zeroSequence = nil
		r.iterAlias = nil
		r.mapAlias = nil
		r.sliceAlias = nil
		r.reflectAlias = nil
		r.typeImports = nil
		copy := r.functionBodies(file).(*ast.File)
		if r.err != nil {
			return nil, r.err
		}

		copy.Decls = append(copy.Decls, r.extraDeclarations...)
		for _, spec := range r.extraImports {
			copy.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}}, copy.Decls...)
		}
		var buf bytes.Buffer
		if err := format.Node(&buf, r.source.checked.Fset, r.renameBindings(copy)); err != nil {
			return nil, err
		}
		output[r.source.checked.Fset.PositionFor(file.Pos(), false).Filename] = buf.Bytes()
	}
	return output, nil
}

// functionBodies visits the same typed syntax tree as the expression inventory.
// It copies the path to each changed function; package expressions do not run here.
func (r *rewrite) functionBodies(node ast.Node) ast.Node {
	if node == nil || !r.contains(node) {
		return node
	}
	switch fn := node.(type) {
	case *ast.ValueSpec:
		copy := *fn
		copy.Values = append([]ast.Expr(nil), fn.Values...)
		for i, value := range fn.Values {
			if r.packageConditional(value) {
				copy.Values[i] = r.packageInitializer(value)
			} else {
				copy.Values[i] = r.functionBodies(value).(ast.Expr)
			}
		}
		return &copy
	case *ast.FuncDecl:
		copy := *fn
		copy.Type, copy.Body = r.function(fn, fn.Type, fn.Body)
		return &copy
	case *ast.FuncLit:
		copy := *fn
		copy.Type, copy.Body = r.function(fn, fn.Type, fn.Body)
		return &copy
	}
	original := reflect.ValueOf(node).Elem()
	copy := reflect.New(original.Type())
	copy.Elem().Set(original)
	for i := 0; i < original.NumField(); i++ {
		field := original.Field(i)
		output := copy.Elem().Field(i)
		if child, ok := field.Interface().(ast.Node); ok {
			if (field.Kind() == reflect.Pointer || field.Kind() == reflect.Interface) && field.IsNil() {
				continue
			}
			output.Set(reflect.ValueOf(r.functionBodies(child)))
		} else if field.Kind() == reflect.Slice {
			// Copy child slices before replacing an element; leave non-AST data alone.
			for j := 0; j < field.Len(); j++ {
				child, ok := field.Index(j).Interface().(ast.Node)
				if !ok || reflect.ValueOf(child).IsNil() {
					continue
				}
				rewritten := r.functionBodies(child)
				if rewritten != child {
					if output.Pointer() == field.Pointer() {
						output.Set(reflect.MakeSlice(field.Type(), field.Len(), field.Len()))
						reflect.Copy(output, field)
					}
					output.Index(j).Set(reflect.ValueOf(rewritten))
				}
			}
		}
	}
	return copy.Interface().(ast.Node)
}
