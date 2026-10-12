package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

type collectionDestination struct {
	index  *ast.Ident
	grow   *ast.Ident
	result *ast.Ident
	typ    types.Type
}

func (r *rewrite) comprehensionExpression(call *ast.CallExpr, record *frontend.Comprehension) expressionResult {
	result := r.name()
	typ := r.typeOf(call)
	destination := &collectionDestination{result: result, typ: typ}
	r.registerComprehension(record, destination)
	r.comprehensionBindings(record.Body)
	empty := &ast.CompositeLit{Type: r.temporaryType(typ, call)}
	declaration := r.resultStorage(result, typ, call, empty)
	output := []ast.Stmt{declaration}
	key, _ := types.ComprehensionTypes(typ)
	if key == nil && r.builtinHidden("append") {
		r.prepareSliceImport()
		count := r.name()
		destination.grow = count
		output = append(output, variable(count, r.temporaryType(types.Typ[types.Int], call), integer(0)))
	}
	if exact := r.exactComprehension(call, record, destination); exact != nil {
		output = exact
	} else {
		output = append(output, r.block(record.Body))
	}
	r.completeExpression(call)
	return r.result(output, []ast.Expr{result})
}

func (r *rewrite) comprehensionOutput(call *ast.CallExpr, destination *collectionDestination) expressionResult {
	r.completeExpression(call)
	beforeEvaluation := r.operands(call.Args)
	before, values := beforeEvaluation.Statements, beforeEvaluation.Expressions()
	keyType, element := types.ComprehensionTypes(destination.typ)
	if keyType == nil {
		value := r.name()
		before = append(before, variable(value, r.temporaryType(element, call.Args[0]), values[0]))
		if destination.index != nil {
			before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{&ast.IndexExpr{X: destination.result,
				Index: destination.index}}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
			return r.result(before, nil)
		}
		if destination.grow != nil {
			return r.result(append(before, r.growComprehension(destination, value)...), nil)
		}
		appendCall := &ast.CallExpr{Fun: ast.NewIdent("append"), Args: []ast.Expr{destination.result, value}}
		before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{destination.result}, Tok: token.ASSIGN,
			Rhs: []ast.Expr{appendCall}})
	} else {
		key, value := r.name(), r.name()
		before = append(before, variable(key, r.temporaryType(keyType, call.Args[0]), values[0]))
		before = append(before, variable(value, r.temporaryType(element, call.Args[1]), values[1]))
		before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{&ast.IndexExpr{X: destination.result,
			Index: key}}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
	}

	return r.result(before, nil)
}

// builtinHidden checks source declarations before emitting a builtin name.
func (r *rewrite) builtinHidden(name string) bool {
	for _, object := range r.input.source.inventory.Checked.TypesInfo.Defs {
		if object != nil && object.Name() == name {
			return true
		}
	}
	for _, object := range r.input.source.inventory.Checked.TypesInfo.Implicits {
		if object != nil && object.Name() == name {
			return true
		}
	}
	return false
}

// comprehensionBindings gives generator variables fresh names. Checked object
// identity keeps their output uses and closures separate from outer bindings.
func (r *rewrite) comprehensionBindings(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncLit); ok && r.realFunction(function) {
			return false
		}
		if loop, ok := node.(*ast.RangeStmt); ok {
			for _, expression := range []ast.Expr{loop.Key, loop.Value} {
				if id, ok := expression.(*ast.Ident); ok && id.Name != "_" {
					r.freshBinding(id)
				}
			}
		}
		return true
	})
}

// growComprehension appends through slices.Grow when source hides append.
func (r *rewrite) growComprehension(destination *collectionDestination, value ast.Expr) []ast.Stmt {
	slicePackage := r.slicePackage()
	result, count := destination.result, destination.grow
	grown := &ast.CallExpr{Fun: &ast.SelectorExpr{X: slicePackage, Sel: ast.NewIdent("Grow")},
		Args: []ast.Expr{result, integer(1)}}
	extended := &ast.SliceExpr{X: result, High: &ast.BinaryExpr{X: count, Op: token.ADD, Y: integer(1)}}
	slot := &ast.IndexExpr{X: result, Index: count}
	return []ast.Stmt{
		&ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN, Rhs: []ast.Expr{grown}},
		&ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN, Rhs: []ast.Expr{extended}},
		&ast.AssignStmt{Lhs: []ast.Expr{slot}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}},
		&ast.IncDecStmt{X: count, Tok: token.INC},
	}
}
