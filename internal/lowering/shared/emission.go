package shared

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"reflect"

	"github.com/lucasavila00/tgo/internal/lowering/source"
)

// CheckLoweredOperations reports any selected operation that was not emitted.
func (r *Context) CheckLoweredOperations() error {
	s := r.Input.Inventory
	for _, site := range s.Sites {
		expression := s.Expressions[source.Site(site)]
		if _, expected := r.Input.Propagations[expression]; expected && !r.Request.Lowered[expression] {
			return fmt.Errorf("%s: propagation was not lowered", s.Checked.Fset.Position(expression.Pos()))
		}
	}
	if err := r.checkLoweredElements(); err != nil {
		return err
	}
	for call := range r.Input.Comprehensions {
		if !r.Request.Lowered[call] {
			return fmt.Errorf("%s: comprehension was not lowered", s.Checked.Fset.Position(call.Pos()))
		}
	}
	for statement := range r.Input.ShortReturns {
		if !r.Request.LoweredReturns[statement] {
			return fmt.Errorf("%s: short return was not lowered", s.Checked.Fset.Position(statement.Pos()))
		}
	}
	for call := range r.Input.Conditionals {
		if !r.Request.Lowered[call] {
			return fmt.Errorf("%s: conditional expression was not lowered", s.Checked.Fset.Position(call.Pos()))
		}
	}
	return nil
}

// Contains reports whether a node contains selected lowering work.
func (r *Context) Contains(node ast.Node) bool {
	if r.functionState.suppressed || node == nil {
		return false
	}
	if contains(node, r.Input.Target) {
		return true
	}
	if r.containsShortReturn(node) {
		return true
	}
	for expression := range r.Input.Propagations {
		if contains(node, expression) {
			return true
		}
	}
	for expression := range r.Input.Comprehensions {
		if contains(node, expression) {
			return true
		}
	}
	if r.containsElement(node) {
		return true
	}
	for expression := range r.Input.Conditionals {
		if contains(node, expression) {
			return true
		}
	}
	return false
}

func (r *Context) isTarget(expression ast.Expr) bool {
	if r.functionState.suppressed {
		return false
	}
	if expression == r.Input.Target && expression != nil {
		return true
	}
	_, ok := r.Input.Propagations[expression]
	call, _ := expression.(*ast.CallExpr)
	return ok || r.Input.Conditionals[call] != nil || r.Input.Comprehensions[call] != nil
}

// Emit returns formatted files without changing the checked input.
func (r *Context) Emit() (map[string][]byte, error) {
	r.Request.names.ReserveSyntax(r.Input.Inventory.Checked.Syntax)
	output := make(map[string][]byte)
	for _, file := range r.Input.Inventory.Checked.Syntax {
		fileRewrite := &Context{Input: r.Input, Request: r.Request, file: &fileState{}, functionState: &functionState{}}
		outputFile := fileRewrite.functionBodies(file).(*ast.File)
		if r.Request.Err != nil {
			return nil, r.Request.Err
		}

		outputFile.Decls = append(outputFile.Decls, fileRewrite.file.extraDeclarations...)
		for _, spec := range fileRewrite.file.extraImports {
			declaration := &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}
			outputFile.Decls = append([]ast.Decl{declaration}, outputFile.Decls...)
		}
		var buf bytes.Buffer
		if err := format.Node(
			&buf, r.Input.Inventory.Checked.Fset, fileRewrite.renameBindings(outputFile),
		); err != nil {
			return nil, err
		}
		output[r.Input.Inventory.Checked.Fset.PositionFor(file.Pos(), false).Filename] = buf.Bytes()
	}
	return output, nil
}

// functionBodies visits the same typed syntax tree as the expression inventory.
// It copies the path to each changed function; package expressions do not run here.
func (r *Context) functionBodies(node ast.Node) ast.Node {
	if node == nil || !r.Contains(node) {
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
	rewriteChildFields(original, rewritten.Elem(), r.functionBodies)
	return rewritten.Interface().(ast.Node)
}

func (r *Context) checkLoweredElements() error {
	s := r.Input.Inventory
	for _, file := range s.Checked.Syntax {
		var missing error
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && s.Checked.Operations[id.Name].Element &&
					!r.Request.Lowered[call] {
					missing = fmt.Errorf("%s: comprehension output was not lowered", s.Checked.Fset.Position(call.Pos()))
				}
			}
			return missing == nil
		})
		if missing != nil {
			return missing
		}
	}
	return nil
}

func (r *Context) containsElement(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(child ast.Node) bool {
		if call, ok := child.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok && r.Input.Inventory.Checked.Operations[id.Name].Element {
				found = true
			}
		}
		return !found
	})
	return found
}

func (r *Context) containsShortReturn(node ast.Node) bool {
	for statement := range r.Input.ShortReturns {
		if contains(node, statement) {
			return true
		}
	}
	return false
}
