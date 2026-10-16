package shared

import (
	"reflect"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
)

// enumTypeSyntax rewrites constant operands in types and keeps label uses outside the declaration.
func (r *Context) enumTypeSyntax(node ast.Node, before *[]ast.Stmt) ast.Node {
	if node == nil || !r.Contains(node) {
		return node
	}
	if expression, ok := node.(ast.Expr); ok {
		if result, handled := r.EnumUnevaluated(expression); handled {
			*before = append(*before, result.Statements...)
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
func (r *Context) constantDeclaration(declaration *ast.GenDecl) []ast.Stmt {
	var before []ast.Stmt
	output := *declaration
	output.Specs = make([]ast.Spec, len(declaration.Specs))
	for index, specification := range declaration.Specs {
		output.Specs[index] = r.constantSpecification(specification.(*ast.ValueSpec), &before)
	}
	return append(before, &ast.DeclStmt{Decl: &output})
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
		evaluation := r.Expression(initializer.Value)
		*before = append(*before, evaluation.Statements...)
		output.Values = append(output.Values, evaluation.Expressions()...)
	}
	return &output
}

// sourceConstantSpecification retains ordinary Go constants with no private checker record.
func (r *Context) sourceConstantSpecification(specification *ast.ValueSpec, before *[]ast.Stmt) *ast.ValueSpec {
	output := *r.enumTypeDeclaration(specification, before).(*ast.ValueSpec)
	evaluation := r.Operands(specification.Values)
	*before = append(*before, evaluation.Statements...)
	output.Values = evaluation.Expressions()
	return &output
}
