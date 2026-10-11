package lowering

import (
	"fmt"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

// After receives references to the evaluated expression results.
type After func(results []ast.Expr) []ast.Stmt

type rewrite struct {
	source            *Source
	target            ast.Expr
	after             After
	names             map[string]bool
	next              int
	inserted          bool
	labels            map[string]string
	bindings          map[types.Object]string
	suppressedRoot    ast.Expr
	propagations      map[ast.Expr]Propagation
	conditionals      map[*ast.CallExpr]*ast.IfStmt
	lowered           map[ast.Expr]bool
	frame             *returnFrame
	formatAlias       *ast.Ident
	suppressed        bool
	extraImports      []*ast.ImportSpec
	extraDeclarations []ast.Decl
	zeroSequence      *ast.Ident
	zeroPointer       *ast.Ident
	iterAlias         *ast.Ident
	mapAlias          *ast.Ident
	sliceAlias        *ast.Ident
	origins           map[ast.Expr]ast.Expr
	parameterAliases  map[*types.TypeParam]*ast.Ident
	aliasStatements   map[ast.Node][]ast.Stmt
	reflectAlias      *ast.Ident
	deferredBases     map[ast.Expr]bool
	temporaryTypes    map[ast.Expr]types.Type
	typeImports       map[*types.Package]*ast.Ident
	err               error
}

func (r *rewrite) name() *ast.Ident {
	return r.nameFor("")
}

// nameFor adds the source name to a fresh generated identifier.
func (r *rewrite) nameFor(sourceName string) *ast.Ident {
	for {
		r.next++
		name := fmt.Sprintf("tgo%d", r.next)
		if sourceName != "" {
			name = sourceName + "_" + name
		}
		if !r.names[name] {
			r.names[name] = true
			return ast.NewIdent(name)
		}
	}
}

func (r *rewrite) save(expr ast.Expr) ([]ast.Stmt, []ast.Expr) {
	original := expr
	if origin := r.origins[expr]; origin != nil {
		original = origin
	}
	count := r.resultCount(original)
	if r.temporaryTypes[expr] != nil {
		count = 1
	}
	if literal, ok := original.(*ast.CompositeLit); ok && literal.Type == nil && r.temporaryTypes[expr] == nil {
		if pair, ok := r.source.parents[original].(*ast.KeyValueExpr); ok && pair.Key == original {
			if _, pointer := r.source.checked.TypesInfo.TypeOf(original).Underlying().(*types.Pointer); !pointer {
				return r.materializeKey(expr.(*ast.CompositeLit), literal)
			}
		}
	}
	typ := r.typeOf(expr)
	if r.temporaryTypes[expr] == nil {
		expr = r.contextLiteral(expr, original)
	}
	if count == 0 {
		return []ast.Stmt{&ast.ExprStmt{X: expr}}, nil
	}
	refs := make([]ast.Expr, count)
	for i := range refs {
		refs[i] = r.name()
	}
	var statements []ast.Stmt
	if r.temporaryTypes == nil {
		r.temporaryTypes = make(map[ast.Expr]types.Type)
	}
	for i, ref := range refs {
		resultType := typ
		if tuple, ok := typ.(*types.Tuple); ok {
			resultType = tuple.At(i).Type()
		} else if i == 1 {
			resultType = types.Typ[types.Bool]
		}
		var value ast.Expr
		if count == 1 {
			value = expr
		}
		statements = append(statements, variable(ref.(*ast.Ident), r.temporaryType(resultType, original), value))
		r.temporaryTypes[ref] = types.Default(resultType)
		if r.origins == nil {
			r.origins = make(map[ast.Expr]ast.Expr)
		}
		r.origins[ref] = original
	}
	if count != 1 {
		statements = append(statements, &ast.AssignStmt{Lhs: refs, Tok: token.ASSIGN, Rhs: []ast.Expr{expr}})
	}
	if count == 1 && r.untypedBoolean(original) {
		return statements, []ast.Expr{&ast.BinaryExpr{X: refs[0], Op: token.EQL, Y: boolean(true)}}
	}
	return statements, refs
}

func (r *rewrite) resultCount(expr ast.Expr) int {
	if tuple, ok := r.source.checked.TypesInfo.TypeOf(expr).(*types.Tuple); ok {
		return tuple.Len()
	}
	if r.source.checked.TypesInfo.Types[expr].HasOk() {
		var node ast.Node = expr
		for {
			if paren, ok := r.source.parents[node].(*ast.ParenExpr); ok {
				node = paren
			} else {
				break
			}
		}
		switch parent := r.source.parents[node].(type) {
		case *ast.AssignStmt:
			if len(parent.Rhs) == 1 && len(parent.Lhs) == 2 {
				return 2
			}
		case *ast.ValueSpec:
			if len(parent.Values) == 1 && len(parent.Names) == 2 {
				return 2
			}
		}
	}
	return 1
}

// operands saves earlier values before a later operand emits statements.
func (r *rewrite) operands(input []ast.Expr) ([]ast.Stmt, []ast.Expr) {
	var statements []ast.Stmt
	var values []ast.Expr
	for _, expr := range input {
		before, refs := r.expression(expr)
		if len(before) != 0 {
			for i, value := range values {
				if r.functionName(value) {
					continue
				}
				if tv, ok := r.source.checked.TypesInfo.Types[value]; ok && (tv.Value != nil || tv.IsType() || tv.IsBuiltin() || tv.IsNil()) {
					continue
				}
				if _, ok := value.(*ast.Ident); ok && r.source.checked.TypesInfo.TypeOf(value) == nil {
					continue
				}
				save, saved := r.save(value)
				statements = append(statements, save...)
				values[i] = saved[0]
			}
			statements = append(statements, before...)
		}
		values = append(values, refs...)
	}
	return statements, values
}

func (r *rewrite) expression(expr ast.Expr) ([]ast.Stmt, []ast.Expr) {
	if expr == nil {
		return nil, nil
	}
	if !r.contains(expr) {
		return nil, []ast.Expr{expr}
	}
	if operation, ok := r.propagations[expr]; ok && !r.suppressed {
		return r.propagatedCall(expr.(*ast.CallExpr), operation)
	}
	if call, ok := expr.(*ast.CallExpr); ok {
		if statement := r.conditionals[call]; statement != nil {
			return r.conditionalCall(call, statement)
		}
	}
	var before []ast.Stmt
	var result ast.Expr = expr
	switch e := expr.(type) {
	case *ast.FuncLit:
		copy := *e
		labels := r.labels
		r.labels = nil
		copy.Type, copy.Body = r.function(e, e.Type, e.Body)
		r.labels = labels
		result = &copy
	case *ast.CallExpr:
		if r.unsafeOperand(e) {
			before, result = r.unevaluatedCall(e)
			break
		}
		copy := *e
		operands := append([]ast.Expr{e.Fun}, e.Args...)
		var values []ast.Expr
		before, values = r.operands(operands)
		copy.Fun, copy.Args = values[0], values[1:]
		result = &copy
	case *ast.ParenExpr:
		copy := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		if len(values) == 1 {
			copy.X = values[0]
			result = &copy
		} else {
			return before, values
		}
	case *ast.UnaryExpr:
		copy := *e
		if e.Op == token.AND {
			before, result = r.address(e.X)
			break
		}
		var values []ast.Expr
		before, values = r.expression(e.X)
		copy.X = values[0]
		result = &copy
	case *ast.BinaryExpr:
		before, result = r.binaryExpression(e)
	case *ast.SelectorExpr:
		before, result = r.selectorExpression(e)
	case *ast.IndexExpr:
		before, result = r.indexExpression(e)
	case *ast.StarExpr:
		copy := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		copy.X = values[0]
		result = &copy
	case *ast.TypeAssertExpr:
		copy := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		copy.X = values[0]
		result = &copy
	case *ast.IndexListExpr:
		copy := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		copy.X = values[0]
		result = &copy
	case *ast.SliceExpr:
		before, result = r.sliceExpression(e)
	case *ast.CompositeLit:
		before, result = r.compositeExpression(e)
	}
	return r.finishExpression(expr, before, result)
}

func (r *rewrite) finishExpression(expr ast.Expr, before []ast.Stmt, result ast.Expr) ([]ast.Stmt, []ast.Expr) {
	if result != expr {
		if r.origins == nil {
			r.origins = make(map[ast.Expr]ast.Expr)
		}
		r.origins[result] = expr
	}
	if expr == r.target && expr != r.suppressedRoot {
		r.inserted = true
		if r.functionName(expr) {
			return append(before, r.after([]ast.Expr{result})...), []ast.Expr{result}
		}
		if tv := r.source.checked.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
			return append(before, r.after([]ast.Expr{result})...), []ast.Expr{result}
		}
		// Read types from the original expression, not from the new tree.
		saved, refs := r.save(result)
		before = append(before, saved...)
		before = append(before, r.after(refs)...)
		return before, refs
	}
	return before, []ast.Expr{result}
}

func (r *rewrite) functionName(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		_, ok := r.source.checked.TypesInfo.Uses[e].(*types.Func)
		return ok
	case *ast.SelectorExpr:
		if r.source.checked.TypesInfo.Selections[e] != nil {
			return false
		}
		_, ok := r.source.checked.TypesInfo.Uses[e.Sel].(*types.Func)
		return ok
	case *ast.IndexExpr:
		return r.source.checked.TypesInfo.Types[e.Index].IsType() && r.functionName(e.X)
	case *ast.IndexListExpr:
		return r.functionName(e.X)
	}
	return false
}

func (r *rewrite) addressableArray(expr ast.Expr) bool {
	array := storageKinds(r.source.checked.TypesInfo.TypeOf(expr)) == arrayStorage
	return array && r.source.checked.TypesInfo.Types[expr].Addressable()
}

func (r *rewrite) arrayBase(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	return r.place(expr)
}

func (r *rewrite) untypedBoolean(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return r.untypedBoolean(e.X)
	case *ast.BinaryExpr:
		switch e.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
			return true
		case token.LAND, token.LOR:
			return r.untypedBoolean(e.X) && r.untypedBoolean(e.Y)
		}
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return r.untypedBoolean(e.X)
		}
	case *ast.Ident:
		if obj := r.source.checked.TypesInfo.Uses[e]; obj != nil {
			if basic, ok := obj.Type().(*types.Basic); ok {
				return basic.Kind() == types.UntypedBool
			}
		}
	}
	return false
}
