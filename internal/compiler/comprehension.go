package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

func comprehensionMarker(
	source *source,
	expression ast.Expr,
) (comprehensionSource, *ast.FuncLit, bool) {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return comprehensionSource{}, nil, false
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok {
		return comprehensionSource{}, nil, false
	}
	metadata, ok := source.Comprehensions[name.Name]
	if !ok {
		return comprehensionSource{}, nil, false
	}
	function, ok := call.Args[0].(*ast.FuncLit)
	return metadata, function, ok
}

// comprehension removes the projection function and keeps its fused loop body.
func (l *propagationLowerer) comprehension(
	metadata comprehensionSource,
	function *ast.FuncLit,
	original ast.Expr,
) (ast.Expr, []ast.Stmt) {
	if function == nil || function.Body == nil {
		l.unit.failAt(metadata.Position, "invalid comprehension projection")
		return original, nil
	}
	l.prepareComprehension(metadata, function.Body)
	statements := l.statements(function.Body.List)
	if len(statements) < 2 {
		l.unit.failAt(metadata.Position, "invalid comprehension projection")
		return original, nil
	}
	last, ok := statements[len(statements)-1].(*ast.ReturnStmt)
	if !ok || len(last.Results) != 1 {
		l.unit.failAt(metadata.Position, "invalid comprehension projection")
		return original, nil
	}
	return last.Results[0], statements[:len(statements)-1]
}

// prepareComprehension binds generated built-ins and selects the proven allocation plan.
func (l *propagationLowerer) prepareComprehension(
	metadata comprehensionSource,
	body *ast.BlockStmt,
) {
	makeCall, outer, ok := l.comprehensionParts(metadata, body)
	if !ok {
		return
	}
	makeCall.Fun = l.unit.generatedUniverse("make", metadata.Position)
	if !metadata.Map {
		if l.optimizeExactSlice(metadata, body, makeCall, outer) {
			return
		}
		l.bindComprehensionAppend(metadata, outer.Body)
	}
	l.addComprehensionCapacity(metadata, body, makeCall, outer)
}

// optimizeExactSlice uses the fixed range indexes when a slice gives the exact result length.
// An identity projection uses Go's bulk copy operation.
func (l *propagationLowerer) optimizeExactSlice(
	metadata comprehensionSource,
	body *ast.BlockStmt,
	makeCall *ast.CallExpr,
	outer *ast.RangeStmt,
) bool {
	if len(makeCall.Args) != 2 || len(body.List) != 3 || len(outer.Body.List) != 1 {
		return false
	}
	sourceType := underlyingSlice(l.unit.info.TypeOf(outer.X))
	outputType := underlyingSlice(l.unit.info.TypeOf(makeCall.Args[0]))
	assignment, appendCall, ok := comprehensionAppend(outer.Body)
	if sourceType == nil || outputType == nil || !ok {
		return false
	}
	source := l.stableComprehensionSource(body, outer)
	makeCall.Args[1] = call(
		l.unit.generatedUniverse("len", metadata.Position),
		source,
	)
	if l.identityComprehension(outer, appendCall, sourceType, outputType) {
		result := assignment.Lhs[0]
		copyStatement := &ast.ExprStmt{X: call(
			l.unit.generatedUniverse("copy", metadata.Position),
			result,
			source,
		)}
		for index, statement := range body.List {
			if statement == outer {
				body.List[index] = copyStatement
				break
			}
		}
		return true
	}
	index, ok := outer.Key.(*ast.Ident)
	if !ok {
		return false
	}
	if index.Name == "_" {
		index = ast.NewIdent(l.unit.freshIdentifier("__tgo_index"))
		outer.Key = index
	}
	assignment.Lhs[0] = &ast.IndexExpr{X: assignment.Lhs[0], Index: index}
	assignment.Rhs[0] = appendCall.Args[1]
	return true
}

func (l *propagationLowerer) stableComprehensionSource(
	body *ast.BlockStmt,
	outer *ast.RangeStmt,
) ast.Expr {
	source := outer.X
	if _, stable := source.(*ast.Ident); stable {
		return source
	}
	name := ast.NewIdent(l.unit.freshIdentifier("__tgo_source"))
	body.List = append([]ast.Stmt{&ast.AssignStmt{
		Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{source},
	}}, body.List...)
	outer.X = name
	return name
}

func (l *propagationLowerer) identityComprehension(
	outer *ast.RangeStmt,
	appendCall *ast.CallExpr,
	source *types.Slice,
	output *types.Slice,
) bool {
	value, valueOK := outer.Value.(*ast.Ident)
	result, resultOK := appendCall.Args[1].(*ast.Ident)
	return valueOK && resultOK &&
		l.unit.info.ObjectOf(value) == l.unit.info.ObjectOf(result) &&
		types.Identical(source.Elem(), output.Elem())
}

func comprehensionAppend(
	body *ast.BlockStmt,
) (*ast.AssignStmt, *ast.CallExpr, bool) {
	if len(body.List) != 1 {
		return nil, nil, false
	}
	assignment, ok := body.List[0].(*ast.AssignStmt)
	if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return nil, nil, false
	}
	appendCall, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || len(appendCall.Args) != 2 {
		return nil, nil, false
	}
	return assignment, appendCall, true
}

func underlyingSlice(value types.Type) *types.Slice {
	if value == nil {
		return nil
	}
	value = types.Unalias(value)
	if named, ok := value.(*types.Named); ok {
		value = named.Underlying()
	}
	slice, _ := value.(*types.Slice)
	return slice
}

func (l *propagationLowerer) comprehensionParts(
	metadata comprehensionSource,
	body *ast.BlockStmt,
) (*ast.CallExpr, *ast.RangeStmt, bool) {
	if len(body.List) < 3 {
		return nil, nil, false
	}
	initialization, ok := body.List[0].(*ast.AssignStmt)
	if !ok || len(initialization.Rhs) != 1 {
		return nil, nil, false
	}
	makeCall, ok := initialization.Rhs[0].(*ast.CallExpr)
	if !ok || len(makeCall.Args) == 0 {
		return nil, nil, false
	}
	if !validComprehensionOutput(l.unit.info.TypeOf(makeCall.Args[0]), metadata.Map) {
		kind := "slice"
		if metadata.Map {
			kind = "map"
		}
		l.unit.failAt(
			metadata.Position,
			"%s comprehension needs a %s output type",
			kind,
			kind,
		)
		return nil, nil, false
	}
	outer, ok := body.List[1].(*ast.RangeStmt)
	if !ok {
		return nil, nil, false
	}
	return makeCall, outer, true
}

func (l *propagationLowerer) bindComprehensionAppend(
	metadata comprehensionSource,
	body *ast.BlockStmt,
) {
	terminal := comprehensionTerminal(body)
	if len(terminal.List) == 1 {
		if assignment, assignmentOK := terminal.List[0].(*ast.AssignStmt); assignmentOK &&
			len(assignment.Rhs) == 1 {
			if appendCall, appendOK := assignment.Rhs[0].(*ast.CallExpr); appendOK {
				appendCall.Fun = l.unit.generatedUniverse("append", metadata.Position)
			}
		}
	}
}

func (l *propagationLowerer) addComprehensionCapacity(
	metadata comprehensionSource,
	body *ast.BlockStmt,
	makeCall *ast.CallExpr,
	outer *ast.RangeStmt,
) {
	_, directResult := outer.Body.List[0].(*ast.AssignStmt)
	if len(makeCall.Args) != 2 || len(body.List) != 3 || len(outer.Body.List) != 1 ||
		!directResult || !lenAllowed(l.unit.info.TypeOf(outer.X)) {
		return
	}
	source := outer.X
	if _, cheap := source.(*ast.Ident); !cheap {
		source = l.stableComprehensionSource(body, outer)
	}
	makeCall.Args = append(makeCall.Args, call(
		l.unit.generatedUniverse("len", metadata.Position),
		source,
	))
}

func comprehensionTerminal(body *ast.BlockStmt) *ast.BlockStmt {
	for len(body.List) == 1 {
		switch nested := body.List[0].(type) {
		case *ast.RangeStmt:
			body = nested.Body
		case *ast.IfStmt:
			body = nested.Body
		default:
			return body
		}
	}
	return body
}

func validComprehensionOutput(value types.Type, mapResult bool) bool {
	if value == nil {
		return false
	}
	value = types.Unalias(value)
	if named, ok := value.(*types.Named); ok {
		value = named.Underlying()
	}
	_, isMap := value.(*types.Map)
	_, isSlice := value.(*types.Slice)
	return mapResult && isMap || !mapResult && isSlice
}

// lenAllowed reports types whose range count has a useful len upper bound.
func lenAllowed(value types.Type) bool {
	if value == nil {
		return false
	}
	value = types.Unalias(value)
	if named, ok := value.(*types.Named); ok {
		return lenAllowed(named.Underlying())
	}
	switch item := value.(type) {
	case *types.Array, *types.Slice, *types.Map:
		return true
	case *types.Pointer:
		_, ok := item.Elem().Underlying().(*types.Array)
		return ok
	case *types.Basic:
		return item.Info()&types.IsString != 0
	default:
		return false
	}
}
