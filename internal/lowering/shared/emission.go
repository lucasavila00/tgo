package shared

import (
	"bytes"
	"fmt"
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/format"
	"go/token"
	"reflect"
)

// CheckLoweredOperations reports any selected operation that was not emitted.
func (r *Context) CheckLoweredOperations() error {
	var missing error
	for _, root := range r.operationRoots() {
		ast.Inspect(root, func(node ast.Node) bool {
			if missing != nil {
				return false
			}
			switch node := node.(type) {
			case *ast.PropagateExpr, *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr:
				if !r.request.lowered[node.(ast.Expr)] {
					missing = fmt.Errorf("%s: TGo expression was not lowered", r.input.Inventory.Checked.Fset.Position(node.Pos()))
				}
			case *ast.ShortReturnStmt:
				if !r.request.loweredReturns[node] {
					missing = fmt.Errorf("%s: short return was not lowered", r.input.Inventory.Checked.Fset.Position(node.Pos()))
				}
			case *ast.CompositeLit:
				if r.input.Inventory.Checked.TypesInfo.EnumConstructors[node] != nil && !r.request.lowered[node] {
					position := r.input.Inventory.Checked.Fset.Position(node.Pos())
					missing = fmt.Errorf("%s: enum construction was not lowered", position)
				}
			}
			return missing == nil
		})
	}
	return missing
}

// operationRoots includes source files and separately checked inherited constant expressions.
func (r *Context) operationRoots() []ast.Node {
	roots := make([]ast.Node, 0, len(r.input.Inventory.Checked.Syntax)+len(r.input.Inventory.ConstantCopies))
	for _, file := range r.input.Inventory.Checked.Syntax {
		roots = append(roots, file)
	}
	for _, expression := range r.input.Inventory.ConstantCopies {
		roots = append(roots, expression)
	}
	return roots
}

// Contains reports whether a node contains selected lowering work.
func (r *Context) Contains(node ast.Node) bool {
	if node == nil {
		return false
	}
	if contains(node, r.input.Target) {
		return true
	}
	if !r.input.TGo {
		return false
	}
	found := false
	ast.Inspect(node, func(child ast.Node) bool {
		switch child.(type) {
		case *ast.PropagateExpr, *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr, *ast.ShortReturnStmt, *ast.EnumType:
			found = true
		}
		if literal, ok := child.(*ast.CompositeLit); ok &&
			r.input.Inventory.Checked.TypesInfo.EnumConstructors[literal] != nil {
			found = true
		}
		if statement, ok := child.(*ast.TypeSwitchStmt); ok &&
			r.input.Inventory.Checked.TypesInfo.EnumSwitches[statement] != nil {
			found = true
		}
		return !found
	})
	return found
}

// isTarget reports whether the expression requires insertion or a TGo rule.
func (r *Context) isTarget(expression ast.Expr) bool {
	if expression != nil && expression == r.input.Target {
		return true
	}
	if !r.input.TGo {
		return false
	}
	if literal, ok := expression.(*ast.CompositeLit); ok &&
		r.input.Inventory.Checked.TypesInfo.EnumConstructors[literal] != nil {
		return true
	}
	switch expression.(type) {
	case *ast.PropagateExpr, *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr:
		return true
	}
	return false
}

// Emit returns formatted files without changing the checked input.
func (r *Context) Emit() (Output, error) {
	r.request.names.ReserveSyntax(r.input.Inventory.Checked.Syntax)
	r.request.names.Reserve(r.input.Inventory.Checked.Types.Scope().Names()...)
	output := make(map[string][]byte)
	for _, file := range r.input.Inventory.Checked.Syntax {
		fileRewrite := &Context{input: r.input, dispatch: r.dispatch, request: r.request,
			file: &fileState{}, functionState: &functionState{}}
		outputFile := fileRewrite.functionBodies(file).(*ast.File)
		outputFile = fileRewrite.enumDeclarations(outputFile)
		if r.request.err != nil {
			return Output{}, r.request.err
		}

		outputFile.Decls = append(outputFile.Decls, fileRewrite.file.extraDeclarations...)
		for _, spec := range fileRewrite.file.extraImports {
			declaration := &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}
			outputFile.Decls = append([]ast.Decl{declaration}, outputFile.Decls...)
		}
		fileRewrite.enumImports(outputFile, file)
		if err := ordinaryOutput(outputFile); err != nil {
			return Output{}, err
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
	if expression, ok := node.(ast.Expr); ok {
		if result, handled := r.EnumUnevaluated(expression); handled {
			return result.Expressions()[0]
		}
	}
	switch fn := node.(type) {
	case *ast.GenDecl:
		if fn.Tok == token.CONST {
			return r.constantDeclaration(fn, nil)
		}
	case *ast.ValueSpec:
		valueSpec := *fn
		if fn.Type != nil {
			valueSpec.Type = r.functionBodies(fn.Type).(ast.Expr)
		}
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
		function.Type = r.functionBodies(fn.Type).(*ast.FuncType)
		function.Body = r.functionBody(fn, fn.Type, fn.Body)
		return &function
	case *ast.FuncLit:
		function := *fn
		function.Type = r.functionBodies(fn.Type).(*ast.FuncType)
		function.Body = r.functionBody(fn, fn.Type, fn.Body)
		return &function
	}
	original := reflect.ValueOf(node).Elem()
	rewritten := reflect.New(original.Type())
	rewritten.Elem().Set(original)
	rewriteChildFields(original, rewritten.Elem(), r.functionBodies)
	return rewritten.Interface().(ast.Node)
}

// ordinaryOutput reports a source-only node left in generated Go.
func ordinaryOutput(node ast.Node) error {
	var invalid error
	ast.Inspect(node, func(child ast.Node) bool {
		switch child.(type) {
		case *ast.PropagateExpr, *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr, *ast.ShortReturnStmt, *ast.EnumType:
			invalid = fmt.Errorf("TGo syntax remains in generated Go: %T", child)
		}
		return invalid == nil
	})
	return invalid
}
