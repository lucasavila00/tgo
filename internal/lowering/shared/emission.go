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
	for _, file := range r.input.Inventory.Checked.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
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
			}
			return missing == nil
		})
	}
	return missing
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
		case *ast.PropagateExpr, *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr, *ast.ShortReturnStmt:
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
	switch expression.(type) {
	case *ast.PropagateExpr, *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr:
		return true
	}
	return false
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

// ordinaryOutput reports a source-only node left in generated Go.
func ordinaryOutput(node ast.Node) error {
	var invalid error
	ast.Inspect(node, func(child ast.Node) bool {
		switch child.(type) {
		case *ast.PropagateExpr, *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr, *ast.ShortReturnStmt:
			invalid = fmt.Errorf("TGo syntax remains in generated Go: %T", child)
		}
		return invalid == nil
	})
	return invalid
}
