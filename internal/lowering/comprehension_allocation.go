package lowering

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

// exactComprehension allocates once when one slice or array range emits one value
// on every visit. It leaves filters, nested ranges, and source jumps on append.
func (r *rewrite) exactComprehension(call *ast.CallExpr, record *frontend.Comprehension, result *ast.Ident,
	typ types.Type) []ast.Stmt {
	key, _ := types.ComprehensionTypes(typ)
	if r.builtinHidden("make") || r.builtinHidden("len") || len(record.Body.List) != 1 {
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
		if function, ok := node.(*ast.FuncLit); ok && r.realFunction(function) {
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
		info := r.input.source.inventory.Checked.TypesInfo
		if !ok || !bound ||
			info.Uses[outputKey] != info.Defs[rangeKey] {
			return nil
		}
	}
	if changesCount {
		return nil
	}
	sourceType := r.typeOf(loop.X).Underlying()
	switch sourceType.(type) {
	case *types.Slice, *types.Array:
	default:
		sourceKey, element := types.ComprehensionTypes(r.typeOf(loop.X))
		if sourceKey != nil || element == nil {
			return nil
		}
	}
	output := *loop
	var before []ast.Stmt
	var count ast.Expr
	if r.input.source.rangeElided(loop) {
		count = &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(sourceType.(*types.Array).Len(), 10)}
	} else {
		before, output.X = r.capture(loop.X)
		count = &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{output.X}}
	}
	allocation := &ast.CallExpr{Fun: ast.NewIdent("make"), Args: []ast.Expr{r.temporaryType(typ, call), count}}
	before = append(before, variable(result, r.temporaryType(typ, call), allocation))
	if key == nil {
		index := r.name()
		if sourceKey, ok := loop.Key.(*ast.Ident); ok && sourceKey.Name != "_" {
			object := r.input.source.inventory.Checked.TypesInfo.Defs[sourceKey]
			saved := variable(index, r.temporaryType(types.Typ[types.Int], call), ast.NewIdent(r.file.bindings[object]))
			output.Body = &ast.BlockStmt{List: append([]ast.Stmt{saved}, loop.Body.List...)}
		} else {
			output.Key = index
			output.Tok = token.DEFINE
		}
		r.functionState.collectionOutputs[record.Output].index = index
	}
	return append(before, &ast.BlockStmt{List: r.rangeStatement(&output)})
}
