package frontend

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

// Marker records a postfix operation and its original source position.
type Marker struct {
	Name     string
	Wrap     bool
	CallName string
	Position token.Position
}

type parsedPostfix struct {
	operand    ast.Expr
	start, end token.Pos
	marker     Marker
}

type postfixBinding struct {
	name string
	end  token.Pos
}

// bindPostfixCalls records source names before any nested operand is replaced.
// It converts parsed suffix positions to input positions for checking and normalization.
func (result *Parsed) bindPostfixCalls(fset *token.FileSet, prepared, original *token.File,
	offsets []int, operations []parsedPostfix) map[ast.Expr]postfixBinding {
	bindings := make(map[ast.Expr]postfixBinding, len(operations))
	for _, operation := range operations {
		call := ast.Unparen(operation.operand).(*ast.CallExpr)
		var name bytes.Buffer
		if err := format.Node(&name, fset, call.Fun); err != nil {
			panic(err)
		}
		marker := operation.marker
		marker.CallName = name.String()
		marker.Position = original.Position(original.Pos(offsets[prepared.Offset(operation.start)]))
		result.Propagations[marker.Name] = marker
		bindings[operation.operand] = postfixBinding{
			name: marker.Name, end: original.Pos(offsets[prepared.Offset(operation.end)-1] + 1),
		}
	}
	return bindings
}

// normalizePostfix replaces recorded operands with checker calls in one bottom-up walk.
// Nested replacements are complete before a parent operand is wrapped.
func normalizePostfix(file *ast.File, bindings map[ast.Expr]postfixBinding) {
	astutil.Apply(file, nil, func(cursor *astutil.Cursor) bool {
		operand, isExpression := cursor.Node().(ast.Expr)
		if !isExpression {
			return true
		}
		binding, found := bindings[operand]
		if found {
			cursor.Replace(&ast.CallExpr{
				Fun:    &ast.Ident{Name: binding.name, NamePos: operand.Pos()},
				Lparen: operand.Pos(), Args: []ast.Expr{ast.Unparen(operand)}, Rparen: binding.end - 1,
			})
		}
		return true
	})
}
