package lowering

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"reflect"

	"github.com/lucasavila00/tgo/internal/lowering/source"
)

// InsertAfter returns Go files with the callback's statements after the selected expression.
// The callback receives saved results; the original syntax tree stays unchanged.
func (s *Source) InsertAfter(site Site, after After) (map[string][]byte, error) {
	target, ok := s.inventory.Expressions[source.Site(site)]
	if !ok {
		return nil, fmt.Errorf("expression site does not belong to this source")
	}
	if site.Reason != "" {
		return nil, fmt.Errorf("expression has no current-function evaluation: %s", site.Reason)
	}
	r := &rewrite{source: s, target: target, after: after}
	output, err := r.emit()
	if err != nil {
		return nil, err
	}
	if !r.inserted {
		return nil, fmt.Errorf("expression site was not lowered: %s:%d", site.File, site.Start)
	}
	return output, nil
}

// LowerPropagations replaces checked TGo operations with ordinary Go.
// It returns formatted file contents by absolute path without writing files.
// It reports an error if any accepted operation was not replaced.
func (s *Source) LowerPropagations(operations map[string]Propagation) (map[string][]byte, error) {
	checked := s.inventory.Checked
	r := &rewrite{
		source:              s,
		propagations:        make(map[ast.Expr]Propagation),
		comprehensions:      checked.Comprehensions,
		comprehensionRanges: make(map[*ast.RangeStmt]bool),
		collectionOutputs:   make(map[string]*collectionDestination),
		conditionals:        checked.Conditionals,
		shortReturns:        checked.ShortReturns,
		loweredReturns:      make(map[*ast.ReturnStmt]bool),
		lowered:             make(map[ast.Expr]bool),
	}
	for site, expression := range s.inventory.Expressions {
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
	if err := r.checkLoweredOperations(); err != nil {
		return nil, err
	}
	return output, nil
}

func (r *rewrite) checkLoweredOperations() error {
	s := r.source
	for _, site := range s.Sites {
		expression := s.inventory.Expressions[source.Site(site)]
		if _, expected := r.propagations[expression]; expected && !r.lowered[expression] {
			return fmt.Errorf("%s: propagation was not lowered", s.inventory.Checked.Fset.Position(expression.Pos()))
		}
	}
	for _, file := range s.inventory.Checked.Syntax {
		var missing error
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && s.inventory.Checked.Operations[id.Name].Element && !r.lowered[call] {
					missing = fmt.Errorf("%s: comprehension output was not lowered", s.inventory.Checked.Fset.Position(call.Pos()))
				}
			}
			return missing == nil
		})
		if missing != nil {
			return missing
		}
	}
	for call := range r.comprehensions {
		if !r.lowered[call] {
			return fmt.Errorf("%s: comprehension was not lowered", s.inventory.Checked.Fset.Position(call.Pos()))
		}
	}
	for statement := range r.shortReturns {
		if !r.loweredReturns[statement] {
			return fmt.Errorf("%s: short return was not lowered", s.inventory.Checked.Fset.Position(statement.Pos()))
		}
	}
	for call := range r.conditionals {
		if !r.lowered[call] {
			return fmt.Errorf("%s: if expression was not lowered", s.inventory.Checked.Fset.Position(call.Pos()))
		}
	}
	return nil
}

func (r *rewrite) contains(node ast.Node) bool {
	if r.suppressed || node == nil {
		return false
	}
	if contains(node, r.target) {
		return true
	}
	for statement := range r.shortReturns {
		if contains(node, statement) {
			return true
		}
	}
	for expression := range r.propagations {
		if contains(node, expression) {
			return true
		}
	}
	for expression := range r.comprehensions {
		if contains(node, expression) {
			return true
		}
	}
	found := false
	ast.Inspect(node, func(child ast.Node) bool {
		if call, ok := child.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok && r.source.inventory.Checked.Operations[id.Name].Element {
				found = true
			}
		}
		return !found
	})
	if found {
		return true
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
	return ok || r.conditionals[call] != nil || r.comprehensions[call] != nil
}

func (r *rewrite) emit() (map[string][]byte, error) {
	r.names.ReserveSyntax(r.source.inventory.Checked.Syntax)
	output := make(map[string][]byte)
	for _, file := range r.source.inventory.Checked.Syntax {
		r.startFile()
		outputFile := r.functionBodies(file).(*ast.File)
		if r.err != nil {
			return nil, r.err
		}

		outputFile.Decls = append(outputFile.Decls, r.file.extraDeclarations...)
		for _, spec := range r.file.extraImports {
			declaration := &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}
			outputFile.Decls = append([]ast.Decl{declaration}, outputFile.Decls...)
		}
		var buf bytes.Buffer
		if err := format.Node(&buf, r.source.inventory.Checked.Fset, r.renameBindings(outputFile)); err != nil {
			return nil, err
		}
		output[r.source.inventory.Checked.Fset.PositionFor(file.Pos(), false).Filename] = buf.Bytes()
	}
	return output, nil
}

func (r *rewrite) startFile() {
	r.file = &fileState{}
}

// functionBodies visits the same typed syntax tree as the expression inventory.
// It copies the path to each changed function; package expressions do not run here.
func (r *rewrite) functionBodies(node ast.Node) ast.Node {
	if node == nil || !r.contains(node) {
		return node
	}
	switch fn := node.(type) {
	case *ast.ValueSpec:
		valueSpec := *fn
		valueSpec.Values = append([]ast.Expr(nil), fn.Values...)
		for i, value := range fn.Values {
			if r.packageConditional(value) {
				valueSpec.Values[i] = r.packageInitializer(value)
			} else {
				valueSpec.Values[i] = r.functionBodies(value).(ast.Expr)
			}
		}
		return &valueSpec
	case *ast.FuncDecl:
		function := *fn
		function.Type, function.Body = r.function(fn, fn.Type, fn.Body)
		return &function
	case *ast.FuncLit:
		function := *fn
		function.Type, function.Body = r.function(fn, fn.Type, fn.Body)
		return &function
	}
	original := reflect.ValueOf(node).Elem()
	rewritten := reflect.New(original.Type())
	rewritten.Elem().Set(original)
	for i := 0; i < original.NumField(); i++ {
		field := original.Field(i)
		output := rewritten.Elem().Field(i)
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
	return rewritten.Interface().(ast.Node)
}
