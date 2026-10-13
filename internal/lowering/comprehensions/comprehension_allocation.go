package comprehensions

import (
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"github.com/lucasavila00/tgo/internal/lowering/source"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

// exactComprehension allocates once when one slice or array range emits one value
// on every visit. It leaves filters, nested ranges, and source jumps on append.
func exactComprehension(r *shared.Context, call *ast.CallExpr, record *frontend.Comprehension,
	destination *shared.CollectionDestination) []ast.Stmt {
	result, typ := destination.Result, destination.Type
	key, _ := types.ComprehensionTypes(typ)
	loop := exactComprehensionLoop(r, record, key)
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
	output := *loop
	var before []ast.Stmt
	var count ast.Expr
	if source.RangeElided(r.Inventory().Checked, loop) {
		count = &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(sourceType.(*types.Array).Len(), 10)}
	} else {
		before, output.X = r.Capture(loop.X)
		count = &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{output.X}}
	}
	allocation := &ast.CallExpr{Fun: ast.NewIdent("make"), Args: []ast.Expr{r.TemporaryType(typ, call), count}}
	before = append(before, shared.Variable(result, r.TemporaryType(typ, call), allocation))
	if key == nil {
		index := r.Name()
		if sourceKey, ok := loop.Key.(*ast.Ident); ok && sourceKey.Name != "_" {
			object := r.Inventory().Checked.TypesInfo.Defs[sourceKey]
			saved := shared.Variable(index, r.TemporaryType(types.Typ[types.Int], call), r.BindingReference(object))
			output.Body = &ast.BlockStmt{List: append([]ast.Stmt{saved}, loop.Body.List...)}
		} else {
			output.Key = index
			output.Tok = token.DEFINE
		}
		destination.Index = index
	}
	return append(before, &ast.BlockStmt{List: r.RangeStatement(&output)})
}

// exactComprehensionLoop finds a single range that emits one item per visit and can use an exact allocation.
// It returns nil when filters, jumps, or hidden builtins prevent this choice.
func exactComprehensionLoop(r *shared.Context, record *frontend.Comprehension, key types.Type) *ast.RangeStmt {
	if r.BuiltinHidden("make") || r.BuiltinHidden("len") || len(record.Body.List) != 1 {
		return nil
	}
	loop, ok := record.Body.List[0].(*ast.RangeStmt)
	if !ok || len(loop.Body.List) != 1 {
		return nil
	}
	statement, ok := loop.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return nil
	}
	leaf, ok := statement.X.(*ast.CallExpr)
	if !ok {
		return nil
	}
	id, ok := leaf.Fun.(*ast.Ident)
	if !ok || id.Name != record.Output {
		return nil
	}
	changesCount := false
	ast.Inspect(loop, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncLit); ok && r.RealFunction(function) {
			return false
		}
		switch node.(type) {
		case *ast.BranchStmt, *ast.ReturnStmt:
			changesCount = true
		}
		return !changesCount
	})
	if key != nil {
		outputKey, ok := ast.Unparen(leaf.Args[0]).(*ast.Ident)
		rangeKey, bound := loop.Key.(*ast.Ident)
		info := r.Inventory().Checked.TypesInfo
		if !ok || !bound ||
			info.Uses[outputKey] != info.Defs[rangeKey] {
			return nil
		}
	}
	if changesCount {
		return nil
	}
	return loop
}
