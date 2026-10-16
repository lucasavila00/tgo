package comprehensions

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"github.com/lucasavila00/tgo/internal/lowering/source"
	"go/token"
	"strconv"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// exactComprehension allocates once when one slice or array range emits one value
// on every visit. It leaves filters, nested ranges, and source jumps on append.
func exactComprehension(r *shared.Context, expression *ast.ComprehensionExpr,
	destination *shared.CollectionDestination) []ast.Stmt {
	result, typ := destination.Result, destination.Type
	key, _ := types.ComprehensionTypes(typ)
	loop := exactComprehensionLoop(r, expression, key)
	if loop == nil {
		return nil
	}
	sourceType := r.TypeOf(loop.X).Underlying()
	switch sourceType.(type) {
	case *types.Slice, *types.Array:
	default:
		sourceKey, element := types.ComprehensionTypes(r.TypeOf(loop.X))
		if sourceKey != nil || element == nil {
			return nil
		}
	}
	output := ast.RangeStmt{For: loop.For, Key: loop.Key, Value: loop.Value,
		TokPos: loop.TokPos, Tok: loop.Tok, Range: loop.Range}
	output.Body = &ast.BlockStmt{Lbrace: loop.For, Rbrace: expression.Rbrace}
	var before []ast.Stmt
	var count ast.Expr
	if source.RangeOperandElided(r.Inventory().Checked, loop.X, loop.Value) {
		evaluation := r.ComprehensionOperand(loop.X, false)
		if len(evaluation.Statements) != 0 {
			panic("elided range operand must not add evaluation statements")
		}
		before, output.X = evaluation.Statements, evaluation.Expressions()[0]
		count = &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(sourceType.(*types.Array).Len(), 10)}
	} else {
		evaluation := r.ComprehensionOperand(loop.X, true)
		before, output.X = evaluation.Statements, evaluation.Expressions()[0]
		count = &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{output.X}}
	}
	allocation := &ast.CallExpr{Fun: ast.NewIdent("make"), Args: []ast.Expr{r.TemporaryType(typ, expression), count}}
	before = append(before, shared.Variable(result, r.TemporaryType(typ, expression), allocation))
	if key == nil {
		index := r.Name()
		if sourceKey, ok := loop.Key.(*ast.Ident); ok && sourceKey.Name != "_" {
			object := r.Inventory().Checked.TypesInfo.Defs[sourceKey]
			saved := shared.Variable(index, r.TemporaryType(types.Typ[types.Int], expression), r.BindingReference(object))
			output.Body.List = []ast.Stmt{saved}
		} else {
			output.Key = index
			output.Tok = token.DEFINE
		}
		destination.Index = index
	}
	output.Body.List = append(output.Body.List, Output(r, outputOperands(expression), destination).Statements...)
	return append(before, &ast.BlockStmt{List: []ast.Stmt{&output}})
}

// exactComprehensionLoop finds a single range that emits one item per visit and can use an exact allocation.
// It returns nil when filters, jumps, or hidden builtins prevent this choice.
func exactComprehensionLoop(r *shared.Context, expression *ast.ComprehensionExpr, key types.Type) *ast.RangeClause {
	if r.BuiltinHidden("make") || r.BuiltinHidden("len") || len(expression.Clauses) != 1 {
		return nil
	}
	loop, ok := expression.Clauses[0].(*ast.RangeClause)
	if !ok {
		return nil
	}
	changesCount := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if _, function := node.(*ast.FuncLit); function {
			return false
		}
		switch node.(type) {
		case *ast.BranchStmt, *ast.ReturnStmt, *ast.ShortReturnStmt:
			changesCount = true
		}
		return !changesCount
	})
	if key != nil {
		operands := outputOperands(expression)
		outputKey, ok := ast.Unparen(operands[0]).(*ast.Ident)
		rangeKey, bound := loop.Key.(*ast.Ident)
		info := r.Inventory().Checked.TypesInfo
		if !ok || !bound || info.Uses[outputKey] != info.Defs[rangeKey] {
			return nil
		}
	}
	if changesCount {
		return nil
	}
	return loop
}
