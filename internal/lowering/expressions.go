package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
	"github.com/lucasavila00/tgo/internal/lowering/names"
	"go/ast"
	"go/token"
)

// After receives references to the evaluated expression results.
type After func(results []ast.Expr) []ast.Stmt

type rewrite struct {
	source              *Source
	target              ast.Expr
	after               After
	names               names.Allocator
	inserted            bool
	labels              map[string]string
	breakExits          map[*ast.BranchStmt]*ast.Ident
	controlTargets      map[ast.Stmt]ast.Stmt
	bindings            map[types.Object]string
	suppressedRoot      ast.Expr
	propagations        map[ast.Expr]Propagation
	comprehensions      map[*ast.CallExpr]*frontend.Comprehension
	comprehensionRanges map[*ast.RangeStmt]bool
	collectionOutputs   map[string]*collectionDestination
	conditionals        map[*ast.CallExpr]*ast.IfStmt
	lowered             map[ast.Expr]bool
	frame               *returnFrame
	formatAlias         *ast.Ident
	suppressed          bool
	extraImports        []*ast.ImportSpec
	extraDeclarations   []ast.Decl
	zeroSequence        *ast.Ident
	iterAlias           *ast.Ident
	mapAlias            *ast.Ident
	sliceAlias          *ast.Ident
	origins             map[ast.Expr]ast.Expr
	parameterAliases    map[*types.TypeParam]*ast.Ident
	aliasStatements     map[ast.Node][]ast.Stmt
	reflectAlias        *ast.Ident
	deferredBases       map[ast.Expr]bool
	temporaryTypes      map[ast.Expr]types.Type
	typeImports         map[*types.Package]*ast.Ident
	err                 error
}

func (r *rewrite) name() *ast.Ident {
	return r.nameFor("")
}

// nameFor adds the source name to a fresh generated identifier.
func (r *rewrite) nameFor(sourceName string) *ast.Ident {
	return r.names.Fresh(sourceName)
}

func (r *rewrite) recordOrigin(expression, original ast.Expr) {
	if r.origins == nil {
		r.origins = make(map[ast.Expr]ast.Expr)
	}
	r.origins[expression] = original
}

func (r *rewrite) recordTemporary(expression ast.Expr, typ types.Type, original ast.Expr) {
	if r.temporaryTypes == nil {
		r.temporaryTypes = make(map[ast.Expr]types.Type)
	}
	r.temporaryTypes[expression] = typ
	r.recordOrigin(expression, original)
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
		if pair, ok := r.source.inventory.Parents[original].(*ast.KeyValueExpr); ok && pair.Key == original {
			if _, pointer := r.source.inventory.Checked.TypesInfo.TypeOf(original).Underlying().(*types.Pointer); !pointer {
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
		r.recordTemporary(ref, types.Default(resultType), original)
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
	if tuple, ok := r.source.inventory.Checked.TypesInfo.TypeOf(expr).(*types.Tuple); ok {
		return tuple.Len()
	}
	if r.source.inventory.Checked.TypesInfo.Types[expr].HasOk() {
		var node ast.Node = expr
		for {
			if paren, ok := r.source.inventory.Parents[node].(*ast.ParenExpr); ok {
				node = paren
			} else {
				break
			}
		}
		switch parent := r.source.inventory.Parents[node].(type) {
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
		operandStatements, operandValues := r.expression(expr)
		if len(operandStatements) != 0 {
			var saveStatements []ast.Stmt
			saveStatements, values = r.savePreviousOperands(values)
			statements = append(statements, saveStatements...)
			statements = append(statements, operandStatements...)
		}
		values = append(values, operandValues...)
	}
	return statements, values
}

func (r *rewrite) savePreviousOperands(values []ast.Expr) ([]ast.Stmt, []ast.Expr) {
	var statements []ast.Stmt
	for i, value := range values {
		if !r.operandNeedsSaving(value) {
			continue
		}
		saveStatements, savedValues := r.save(value)
		statements = append(statements, saveStatements...)
		values[i] = savedValues[0]
	}
	return statements, values
}

func (r *rewrite) operandNeedsSaving(value ast.Expr) bool {
	if r.functionName(value) {
		return false
	}
	if typeAndValue, ok := r.source.inventory.Checked.TypesInfo.Types[value]; ok {
		if typeAndValue.Value != nil || typeAndValue.IsType() || typeAndValue.IsBuiltin() || typeAndValue.IsNil() {
			return false
		}
	}
	if _, identifier := value.(*ast.Ident); identifier && r.source.inventory.Checked.TypesInfo.TypeOf(value) == nil {
		return false
	}
	return true
}

func (r *rewrite) expression(expr ast.Expr) ([]ast.Stmt, []ast.Expr) {
	if expr == nil {
		return nil, nil
	}
	if !r.contains(expr) {
		return nil, []ast.Expr{expr}
	}
	if statements, values, special := r.specialExpression(expr); special {
		return statements, values
	}
	if parenthesized, ok := expr.(*ast.ParenExpr); ok {
		statements, values := r.expression(parenthesized.X)
		if len(values) != 1 {
			return statements, values
		}
		output := *parenthesized
		output.X = values[0]
		return r.finishExpression(expr, statements, &output)
	}
	statements, result := r.expressionSyntax(expr)
	return r.finishExpression(expr, statements, result)
}

func (r *rewrite) specialExpression(expr ast.Expr) ([]ast.Stmt, []ast.Expr, bool) {
	if operation, ok := r.propagations[expr]; ok && !r.suppressed {
		statements, values := r.propagatedCall(expr.(*ast.CallExpr), operation)
		return statements, values, true
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}
	if record := r.comprehensions[call]; record != nil {
		statements, values := r.comprehensionCall(call, record)
		return statements, values, true
	}
	if identifier, ok := call.Fun.(*ast.Ident); ok {
		if destination := r.collectionOutputs[identifier.Name]; destination != nil {
			statements, values := r.comprehensionOutput(call, destination)
			return statements, values, true
		}
	}
	if statement := r.conditionals[call]; statement != nil {
		statements, values := r.conditionalCall(call, statement)
		return statements, values, true
	}
	return nil, nil, false
}

func (r *rewrite) expressionSyntax(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr = expr
	switch e := expr.(type) {
	case *ast.FuncLit:
		function := *e
		labels := r.labels
		r.labels = nil
		function.Type, function.Body = r.function(e, e.Type, e.Body)
		r.labels = labels
		result = &function
	case *ast.CallExpr:
		if r.unsafeOperand(e) {
			before, result = r.unevaluatedCall(e)
			break
		}
		call := *e
		operands := append([]ast.Expr{e.Fun}, e.Args...)
		var values []ast.Expr
		before, values = r.operands(operands)
		call.Fun = values[0]
		call.Args = values[1:]
		result = &call
	case *ast.UnaryExpr:
		unary := *e
		if e.Op == token.AND {
			before, result = r.address(e.X)
			break
		}
		var values []ast.Expr
		before, values = r.expression(e.X)
		unary.X = values[0]
		result = &unary
	case *ast.BinaryExpr:
		before, result = r.binaryExpression(e)
	case *ast.SelectorExpr:
		before, result = r.selectorExpression(e)
	case *ast.IndexExpr:
		before, result = r.indexExpression(e)
	case *ast.StarExpr:
		dereference := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		dereference.X = values[0]
		result = &dereference
	case *ast.TypeAssertExpr:
		assertion := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		assertion.X = values[0]
		result = &assertion
	case *ast.IndexListExpr:
		indexList := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		indexList.X = values[0]
		result = &indexList
	case *ast.SliceExpr:
		before, result = r.sliceExpression(e)
	case *ast.CompositeLit:
		before, result = r.compositeExpression(e)
	}
	return before, result
}

func (r *rewrite) finishExpression(expr ast.Expr, before []ast.Stmt, result ast.Expr) ([]ast.Stmt, []ast.Expr) {
	if result != expr {
		r.recordOrigin(result, expr)
	}
	if expr == r.target && expr != r.suppressedRoot {
		r.inserted = true
		if r.functionName(expr) {
			return append(before, r.after([]ast.Expr{result})...), []ast.Expr{result}
		}
		if tv := r.source.inventory.Checked.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
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
		_, ok := r.source.inventory.Checked.TypesInfo.Uses[e].(*types.Func)
		return ok
	case *ast.SelectorExpr:
		if r.source.inventory.Checked.TypesInfo.Selections[e] != nil {
			return false
		}
		_, ok := r.source.inventory.Checked.TypesInfo.Uses[e.Sel].(*types.Func)
		return ok
	case *ast.IndexExpr:
		return r.source.inventory.Checked.TypesInfo.Types[e.Index].IsType() && r.functionName(e.X)
	case *ast.IndexListExpr:
		return r.functionName(e.X)
	}
	return false
}

func (r *rewrite) addressableArray(expr ast.Expr) bool {
	array := storageKinds(r.source.inventory.Checked.TypesInfo.TypeOf(expr)) == arrayStorage
	return array && r.source.inventory.Checked.TypesInfo.Types[expr].Addressable()
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
		if obj := r.source.inventory.Checked.TypesInfo.Uses[e]; obj != nil {
			if basic, ok := obj.Type().(*types.Basic); ok {
				return basic.Kind() == types.UntypedBool
			}
		}
	}
	return false
}
