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
	s := r.input.Inventory
	for _, site := range s.Sites {
		expression := s.Expressions[source.Site(site)]
		if _, expected := r.input.Propagations[expression]; expected && !r.request.lowered[expression] {
			return fmt.Errorf("%s: propagation was not lowered", s.Checked.Fset.Position(expression.Pos()))
		}
	}
	if err := r.checkLoweredElements(); err != nil {
		return err
	}
	for call := range r.input.Comprehensions {
		if !r.request.lowered[call] {
			return fmt.Errorf("%s: comprehension was not lowered", s.Checked.Fset.Position(call.Pos()))
		}
	}
	for statement := range r.input.ShortReturns {
		if !r.request.loweredReturns[statement] {
			return fmt.Errorf("%s: short return was not lowered", s.Checked.Fset.Position(statement.Pos()))
		}
	}
	for call := range r.input.Conditionals {
		if !r.request.lowered[call] {
			return fmt.Errorf("%s: conditional expression was not lowered", s.Checked.Fset.Position(call.Pos()))
		}
	}
	return nil
}

// Contains reports whether a node contains selected lowering work.
func (r *Context) Contains(node ast.Node) bool {
	if node == nil {
		return false
	}
	if contains(node, r.input.Target) {
		return true
	}
	if r.containsShortReturn(node) {
		return true
	}
	for expression := range r.input.Propagations {
		if contains(node, expression) {
			return true
		}
	}
	for expression := range r.input.Comprehensions {
		if contains(node, expression) {
			return true
		}
	}
	if r.containsElement(node) {
		return true
	}
	for expression := range r.input.Conditionals {
		if contains(node, expression) {
			return true
		}
	}
	return false
}

// isTarget reports whether the expression is an insertion target or a recorded TGo expression.
func (r *Context) isTarget(expression ast.Expr) bool {
	if expression == r.input.Target && expression != nil {
		return true
	}
	_, ok := r.input.Propagations[expression]
	call, _ := expression.(*ast.CallExpr)
	return ok || r.input.Conditionals[call] != nil || r.input.Comprehensions[call] != nil
}

// Emit returns formatted files without changing the checked input.
func (r *Context) Emit() (Output, error) {
	r.request.names.ReserveSyntax(r.input.Inventory.Checked.Syntax)
	output := make(map[string][]byte)
	for _, file := range r.input.Inventory.Checked.Syntax {
		fileRewrite := &Context{input: r.input, dispatch: r.dispatch, request: r.request,
			file: &fileState{}, functionState: &functionState{}}
		outputFile := fileRewrite.functionBodies(file).(*ast.File)
		if r.request.err != nil {
			return Output{}, r.request.err
		}

		outputFile.Decls = append(outputFile.Decls, fileRewrite.file.extraDeclarations...)
		for _, spec := range fileRewrite.file.extraImports {
			declaration := &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}
			outputFile.Decls = append([]ast.Decl{declaration}, outputFile.Decls...)
		}
		var buf bytes.Buffer
		if err := format.Node(
			&buf, r.input.Inventory.Checked.Fset, fileRewrite.renameBindings(outputFile),
		); err != nil {
			return Output{}, err
		}
		output[r.input.Inventory.Checked.Fset.PositionFor(file.Pos(), false).Filename] = buf.Bytes()
	}
	return Output{Files: output, Inserted: r.request.inserted}, nil
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
		function.Body = r.functionBody(fn, fn.Type, fn.Body)
		return &function
	case *ast.FuncLit:
		function := *fn
		function.Body = r.functionBody(fn, fn.Type, fn.Body)
		return &function
	}
	original := reflect.ValueOf(node).Elem()
	rewritten := reflect.New(original.Type())
	rewritten.Elem().Set(original)
	rewriteChildFields(original, rewritten.Elem(), r.functionBodies)
	return rewritten.Interface().(ast.Node)
}

// checkLoweredElements reports a comprehension output marker that remains absent from the emitted-operation records.
func (r *Context) checkLoweredElements() error {
	s := r.input.Inventory
	for _, file := range s.Checked.Syntax {
		var missing error
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && s.Checked.Operations[id.Name].Element &&
					!r.request.lowered[call] {
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

// containsElement reports whether syntax contains a checked comprehension output marker.
func (r *Context) containsElement(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(child ast.Node) bool {
		if call, ok := child.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok && r.input.Inventory.Checked.Operations[id.Name].Element {
				found = true
			}
		}
		return !found
	})
	return found
}

// containsShortReturn reports whether the source range contains a checked short return.
func (r *Context) containsShortReturn(node ast.Node) bool {
	for statement := range r.input.ShortReturns {
		if contains(node, statement) {
			return true
		}
	}
	return false
}
