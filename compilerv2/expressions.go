package compilerv2

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// After receives references to the evaluated expression results.
type After func(results []ast.Expr) []ast.Stmt

type rewrite struct {
	source   *Source
	target   ast.Expr
	after    After
	names    map[string]bool
	next     int
	inserted bool
	labels   map[string]string
}

func (r *rewrite) name() *ast.Ident {
	for {
		r.next++
		name := fmt.Sprintf("tgo%d", r.next)
		if !r.names[name] {
			r.names[name] = true
			return ast.NewIdent(name)
		}
	}
}

func (r *rewrite) save(expr ast.Expr) ([]ast.Stmt, []ast.Expr) {
	count := r.resultCount(expr)
	if count == 0 {
		return []ast.Stmt{&ast.ExprStmt{X: expr}}, nil
	}
	refs := make([]ast.Expr, count)
	for i := range refs {
		refs[i] = r.name()
	}
	statement := &ast.AssignStmt{Lhs: refs, Tok: token.DEFINE, Rhs: []ast.Expr{expr}}
	if count == 1 && r.untypedBoolean(expr) {
		return []ast.Stmt{statement}, []ast.Expr{&ast.BinaryExpr{X: refs[0], Op: token.EQL, Y: boolean(true)}}
	}
	return []ast.Stmt{statement}, refs
}

func (r *rewrite) resultCount(expr ast.Expr) int {
	if tuple, ok := r.source.Package.TypesInfo.TypeOf(expr).(*types.Tuple); ok {
		return tuple.Len()
	}
	if r.source.Package.TypesInfo.Types[expr].HasOk() {
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
				if tv, ok := r.source.Package.TypesInfo.Types[value]; ok && (tv.Value != nil || tv.IsType() || tv.IsBuiltin() || tv.IsNil()) {
					continue
				}
				if _, ok := value.(*ast.Ident); ok && r.source.Package.TypesInfo.TypeOf(value) == nil {
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
	if !contains(expr, r.target) {
		return nil, []ast.Expr{expr}
	}
	var before []ast.Stmt
	var result ast.Expr = expr
	switch e := expr.(type) {
	case *ast.FuncLit:
		copy := *e
		labels := r.labels
		r.labels = nil
		copy.Body = r.block(e.Body)
		r.labels = labels
		result = &copy
	case *ast.CallExpr:
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
			before, copy.X = r.place(e.X)
			result = &copy
			break
		}
		var values []ast.Expr
		before, values = r.expression(e.X)
		copy.X = values[0]
		result = &copy
	case *ast.BinaryExpr:
		copy := *e
		if e.Op == token.LAND || e.Op == token.LOR {
			leftBefore, left := r.expression(e.X)
			rightBefore, right := r.expression(e.Y)
			before = leftBefore
			copy.X, copy.Y = left[0], right[0]
			if len(rightBefore) == 0 {
				result = &copy
				break
			}
			value := r.name()
			// The false operand gives the local the original result type without evaluating e.
			zero := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.INT, Value: "0"}, Op: token.NEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}}
			before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.BinaryExpr{X: zero, Op: token.LAND, Y: e}}})
			before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.ASSIGN, Rhs: left})
			var condition ast.Expr = value
			if e.Op == token.LOR {
				condition = &ast.UnaryExpr{Op: token.NOT, X: value}
			}
			rightBefore = append(rightBefore, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.ASSIGN, Rhs: right})
			before = append(before, &ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: rightBefore}})
			result = value
			if r.untypedBoolean(e) {
				result = &ast.BinaryExpr{X: value, Op: token.EQL, Y: boolean(true)}
			}
		} else {
			var values []ast.Expr
			before, values = r.operands([]ast.Expr{e.X, e.Y})
			copy.X, copy.Y = values[0], values[1]
			result = &copy
		}
	case *ast.SelectorExpr:
		copy := *e
		if selection := r.source.Package.TypesInfo.Selections[e]; selection != nil && selection.Kind() == types.MethodVal {
			signature := selection.Obj().Type().(*types.Signature)
			_, pointerReceiver := signature.Recv().Type().Underlying().(*types.Pointer)
			_, pointerValue := r.source.Package.TypesInfo.TypeOf(e.X).Underlying().(*types.Pointer)
			if pointerReceiver && !pointerValue && r.source.Package.TypesInfo.Types[e.X].Addressable() {
				var place ast.Expr
				before, place = r.place(e.X)
				copy.X = &ast.ParenExpr{X: &ast.UnaryExpr{Op: token.AND, X: place}}
				result = &copy
				break
			}
		}
		var values []ast.Expr
		before, values = r.expression(e.X)
		copy.X = values[0]
		result = &copy
	case *ast.IndexExpr:
		copy := *e
		if r.addressableArray(e.X) && contains(e.Index, r.target) {
			var base ast.Expr
			before, base = r.arrayBase(e.X)
			setup, index := r.expression(e.Index)
			before = append(before, setup...)
			copy.X, copy.Index = base, index[0]
			result = &copy
			break
		}
		var values []ast.Expr
		before, values = r.operands([]ast.Expr{e.X, e.Index})
		copy.X, copy.Index = values[0], values[1]
		result = &copy
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
		copy := *e
		input := []ast.Expr{e.X}
		array := r.addressableArray(e.X)
		if array {
			before, copy.X = r.arrayBase(e.X)
			input = nil
		}
		for _, bound := range []ast.Expr{e.Low, e.High, e.Max} {
			if bound != nil {
				input = append(input, bound)
			}
		}
		var values []ast.Expr
		setup, values := r.operands(input)
		before = append(before, setup...)
		i := 0
		if !array {
			copy.X = values[0]
			i = 1
		}
		for _, bound := range []*ast.Expr{&copy.Low, &copy.High, &copy.Max} {
			if *bound != nil {
				*bound = values[i]
				i++
			}
		}
		result = &copy
	case *ast.CompositeLit:
		copy := *e
		var input []ast.Expr
		for _, element := range e.Elts {
			if pair, ok := element.(*ast.KeyValueExpr); ok {
				if tv := r.source.Package.TypesInfo.Types[pair.Key]; tv.IsValue() {
					input = append(input, pair.Key)
				}
				input = append(input, pair.Value)
			} else {
				input = append(input, element)
			}
		}
		var values []ast.Expr
		before, values = r.operands(input)
		copy.Elts = make([]ast.Expr, len(e.Elts))
		i := 0
		for j, element := range e.Elts {
			if pair, ok := element.(*ast.KeyValueExpr); ok {
				kv := *pair
				if tv := r.source.Package.TypesInfo.Types[pair.Key]; tv.IsValue() {
					kv.Key = values[i]
					i++
				}
				kv.Value = values[i]
				i++
				copy.Elts[j] = &kv
			} else {
				copy.Elts[j] = values[i]
				i++
			}
		}
		result = &copy
	}
	if expr == r.target {
		r.inserted = true
		if r.functionName(expr) {
			return append(before, r.after([]ast.Expr{result})...), []ast.Expr{result}
		}
		if tv := r.source.Package.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
			return append(before, r.after([]ast.Expr{result})...), []ast.Expr{result}
		}
		// Read types from the original expression, not from the new tree.
		count := r.resultCount(expr)
		refs := make([]ast.Expr, count)
		for i := range refs {
			refs[i] = r.name()
		}
		if count == 0 {
			before = append(before, &ast.ExprStmt{X: result})
		} else {
			before = append(before, &ast.AssignStmt{Lhs: refs, Tok: token.DEFINE, Rhs: []ast.Expr{result}})
		}
		before = append(before, r.after(refs)...)
		if count == 1 && r.untypedBoolean(expr) {
			refs = []ast.Expr{&ast.BinaryExpr{X: refs[0], Op: token.EQL, Y: boolean(true)}}
		}
		return before, refs
	}
	return before, []ast.Expr{result}
}

func (r *rewrite) functionName(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		_, ok := r.source.Package.TypesInfo.Uses[e].(*types.Func)
		return ok
	case *ast.SelectorExpr:
		if r.source.Package.TypesInfo.Selections[e] != nil {
			return false
		}
		_, ok := r.source.Package.TypesInfo.Uses[e.Sel].(*types.Func)
		return ok
	case *ast.IndexExpr:
		return r.source.Package.TypesInfo.Types[e.Index].IsType() && r.functionName(e.X)
	case *ast.IndexListExpr:
		return r.functionName(e.X)
	}
	return false
}

func (r *rewrite) addressableArray(expr ast.Expr) bool {
	_, array := r.source.Package.TypesInfo.TypeOf(expr).Underlying().(*types.Array)
	return array && r.source.Package.TypesInfo.Types[expr].Addressable()
}

func (r *rewrite) arrayBase(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	before, place := r.place(expr)
	saved, values := r.save(&ast.UnaryExpr{Op: token.AND, X: place})
	return append(before, saved...), &ast.ParenExpr{X: &ast.StarExpr{X: values[0]}}
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
		if obj := r.source.Package.TypesInfo.Uses[e]; obj != nil {
			if basic, ok := obj.Type().(*types.Basic); ok {
				return basic.Kind() == types.UntypedBool
			}
		}
	}
	return false
}
