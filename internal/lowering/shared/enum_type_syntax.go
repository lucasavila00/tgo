package shared

import (
	"reflect"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
)

// enumTypeSyntax keeps constant syntax and lowers nested functions and unevaluated native operands.
// Its only statement output is a reference to an outside label, which belongs before a local declaration.
func (r *Context) enumTypeSyntax(node ast.Node, before *[]ast.Stmt) ast.Node {
	if node == nil || !r.Contains(node) {
		return node
	}
	if expression, ok := node.(ast.Expr); ok {
		if result, handled := r.constantUnevaluated(expression); handled {
			if len(result.Statements) != 0 {
				*before = append(*before, result.Statements...)
			}
			return result.Expressions()[0]
		}
	}
	if _, function := node.(*ast.FuncLit); function {
		return r.functionBodies(node)
	}
	original := reflect.ValueOf(node).Elem()
	output := reflect.New(original.Type())
	output.Elem().Set(original)
	rewriteChildFields(original, output.Elem(), func(child ast.Node) ast.Node {
		return r.enumTypeSyntax(child, before)
	})
	return output.Interface().(ast.Node)
}

// enumTypeDeclaration rewrites only a declaration's type; its values keep normal evaluation order.
func (r *Context) enumTypeDeclaration(specification ast.Spec, before *[]ast.Stmt) ast.Spec {
	switch specification := specification.(type) {
	case *ast.TypeSpec:
		output := *specification
		output.Type = r.enumTypeSyntax(specification.Type, before).(ast.Expr)
		return &output
	case *ast.ValueSpec:
		if specification.Type != nil {
			output := *specification
			output.Type = r.enumTypeSyntax(specification.Type, before).(ast.Expr)
			return &output
		}
	}
	return specification
}

// constantDeclaration keeps each checked initializer and its original iota position.
// Package constants pass nil: only local constants can refer to an outside label.
func (r *Context) constantDeclaration(declaration *ast.GenDecl, before *[]ast.Stmt) *ast.GenDecl {
	output := *declaration
	output.Specs = make([]ast.Spec, len(declaration.Specs))
	for index, specification := range declaration.Specs {
		output.Specs[index] = r.constantSpecification(specification.(*ast.ValueSpec), before)
	}
	return &output
}

// constantSpecification writes inherited values from their own checked syntax.
func (r *Context) constantSpecification(specification *ast.ValueSpec, before *[]ast.Stmt) *ast.ValueSpec {
	output := *specification
	output.Values = nil
	for index, name := range specification.Names {
		object := r.input.Inventory.Checked.TypesInfo.Defs[name].(*types.Const)
		initializer, recorded := r.input.Inventory.Checked.TypesInfo.ConstInitializers[object]
		if !recorded {
			return r.sourceConstantSpecification(specification, before)
		}
		if initializer.Type != nil {
			typ := r.enumTypeSyntax(initializer.Type, before).(ast.Expr)
			if index == 0 {
				output.Type = typ
			}
		}
		output.Values = append(output.Values, r.enumTypeSyntax(initializer.Value, before).(ast.Expr))
	}
	return &output
}

// sourceConstantSpecification retains ordinary Go constants with no private checker record.
func (r *Context) sourceConstantSpecification(specification *ast.ValueSpec, before *[]ast.Stmt) *ast.ValueSpec {
	output := *r.enumTypeDeclaration(specification, before).(*ast.ValueSpec)
	output.Values = make([]ast.Expr, len(specification.Values))
	for index, value := range specification.Values {
		output.Values[index] = r.enumTypeSyntax(value, before).(ast.Expr)
	}
	return &output
}

// constantUnevaluated removes native operations that a checked constant builtin does not execute.
func (r *Context) constantUnevaluated(expression ast.Expr) (Result, bool) {
	call, ok := expression.(*ast.CallExpr)
	if ok && r.unsafeOperand(call) && r.input.Inventory.Checked.TypesInfo.Types[call].Value != nil &&
		r.hasDirectNativeExpression(call.Args[0]) {
		return r.enumUnsafeConstant(call), true
	}
	return r.EnumUnevaluated(expression)
}

// hasDirectNativeExpression excludes function bodies, which keep their own statement lowering.
func (r *Context) hasDirectNativeExpression(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if _, function := node.(*ast.FuncLit); function {
			return false
		}
		if expression, ok := node.(ast.Expr); ok && r.isTarget(expression) {
			found = true
		}
		return !found
	})
	return found
}
