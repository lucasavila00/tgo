package compiler

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
)

func (p *packageUnit) zeroValid(t types.Type) bool {
	return p.zero(t, map[types.Type]bool{})
}
func (p *packageUnit) zero(t types.Type, seen map[types.Type]bool) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if seen[t] {
		return true
	}
	seen[t] = true
	if _, m := p.modelForType(t); m != nil && m.requiresConstructor() {
		return false
	}
	switch x := t.(type) {
	case *types.Named:
		return p.zero(x.Underlying(), seen)
	case *types.Basic, *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature:
		return true
	case *types.Struct:
		for i := 0; i < x.NumFields(); i++ {
			if !p.zero(x.Field(i).Type(), seen) {
				return false
			}
		}
		return true
	case *types.Array:
		return x.Len() == 0 || p.zero(x.Elem(), seen)
	case *types.Interface:
		return x.IsMethodSet()
	case *types.TypeParam:
		return p.constraintZero(x.Constraint(), seen)
	}
	return false
}
func (p *packageUnit) constraintZero(t types.Type, seen map[types.Type]bool) bool {
	iface, ok := t.Underlying().(*types.Interface)
	if !ok {
		return p.zero(t, seen)
	}
	// One bounded embedded term is enough to limit the intersection.
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		embedded := iface.EmbeddedType(i)
		if union, ok := embedded.(*types.Union); ok {
			valid := true
			for j := 0; j < union.Len(); j++ {
				if !p.zero(union.Term(j).Type(), map[types.Type]bool{}) {
					valid = false
				}
			}
			if valid {
				return true
			}
		} else if nested, ok := embedded.Underlying().(*types.Interface); ok {
			if p.constraintZero(nested, seen) {
				return true
			}
		} else if p.zero(embedded, seen) {
			return true
		}
	}
	return false
}
func integer(info *types.Info, e ast.Expr) (int64, bool) {
	if e == nil {
		return 0, false
	}
	tv := info.Types[e]
	if tv.Value == nil || tv.Value.Kind() != constant.Int {
		return 0, false
	}
	return constant.Int64Val(tv.Value)
}
func ident(e ast.Expr, name string) bool { id, ok := e.(*ast.Ident); return ok && id.Name == name }

func (p *packageUnit) checkRules() {
	for _, source := range p.Sources {
		for _, declaration := range source.File.Decls {
			if p.generatedDecl(declaration) {
				continue
			}
			parents := parentNodes(declaration)
			ast.Inspect(declaration, func(node ast.Node) bool {
				p.checkNode(node, parents)
				return true
			})
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Body != nil {
				p.checkResults(function)
			}
		}
	}
}

func parentNodes(root ast.Node) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return parents
}

func (p *packageUnit) checkNode(node ast.Node, parents map[ast.Node]ast.Node) {
	switch node := node.(type) {
	case *ast.ValueSpec:
		declaration, ok := parents[node].(*ast.GenDecl)
		repeatedConstant := ok && declaration.Tok == token.CONST
		if len(node.Values) == 0 && !repeatedConstant {
			p.fail(node, "variables need an initializer")
		}
	case *ast.CompositeLit:
		p.checkLiteral(node)
	case *ast.CallExpr:
		p.checkCall(node)
	case *ast.SelectorExpr:
		p.checkSelector(node)
	case *ast.SliceExpr:
		p.checkSlice(node, parents)
	case *ast.IndexExpr:
		p.checkMapRead(node, parents)
	case *ast.UnaryExpr:
		if node.Op == token.ARROW && !p.zeroValid(firstResult(p.info.TypeOf(node))) {
			p.checkPresence(node, parents)
		}
	case *ast.TypeAssertExpr:
		if node.Type != nil && !p.zeroValid(firstResult(p.info.TypeOf(node))) {
			p.checkPresence(node, parents)
		}
	}
}

func (p *packageUnit) checkSelector(selector *ast.SelectorExpr) {
	if selector.Sel.Pos() == token.NoPos {
		return
	}
	typ := p.info.TypeOf(selector.X)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	_, model := p.modelForType(typ)
	if model == nil || !model.requiresConstructor() {
		return
	}
	selection := p.info.Selections[selector]
	if selection != nil && selection.Kind() == types.FieldVal {
		p.fail(selector, "%s representation is private; use constructors and match", model.Name)
	}
	if strings.HasPrefix(selector.Sel.Name, "Tgo") {
		p.fail(selector, "use match to read an enum payload")
	}
}

func (p *packageUnit) checkMapRead(index *ast.IndexExpr, parents map[ast.Node]ast.Node) {
	typ := p.info.TypeOf(index.X)
	if typ == nil {
		return
	}
	mapping, ok := typ.Underlying().(*types.Map)
	if !ok || p.zeroValid(mapping.Elem()) {
		return
	}
	if assignment, ok := parents[index].(*ast.AssignStmt); ok {
		for _, left := range assignment.Lhs {
			if left == index {
				return
			}
		}
	}
	p.checkPresence(index, parents)
}

func (p *packageUnit) checkLiteral(lit *ast.CompositeLit) {
	t := p.info.TypeOf(lit)
	if t == nil {
		return
	}
	if _, m := p.modelForType(t); m != nil && m.requiresConstructor() {
		p.fail(lit, "use a constructor for %s", m.Name)
		return
	}
	switch typ := t.Underlying().(type) {
	case *types.Struct:
		fields := map[string]bool{}
		keyed := false
		for _, e := range lit.Elts {
			if kv, ok := e.(*ast.KeyValueExpr); ok {
				keyed = true
				if id, ok := kv.Key.(*ast.Ident); ok {
					fields[id.Name] = true
				}
			}
		}
		if !keyed && len(lit.Elts) == typ.NumFields() {
			return
		}
		for i := 0; i < typ.NumFields(); i++ {
			if !fields[typ.Field(i).Name()] {
				p.fail(lit, "missing required field %s", typ.Field(i).Name())
			}
		}
	case *types.Array:
		p.checkElements(lit, typ.Len())
	case *types.Slice:
		p.checkElements(lit, -1)
	}
}
func (p *packageUnit) checkElements(lit *ast.CompositeLit, length int64) {
	supplied := map[int64]bool{}
	next := int64(0)
	largest := int64(-1)
	for _, e := range lit.Elts {
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			value, ok := integer(p.info, kv.Key)
			if !ok {
				return
			}
			next = value
		}
		supplied[next] = true
		if next > largest {
			largest = next
		}
		next++
	}
	if length < 0 {
		length = largest + 1
	}
	if int64(len(supplied)) != length {
		p.fail(lit, "array and slice literals must supply every index")
	}
}
func (p *packageUnit) checkCall(c *ast.CallExpr) {
	if p.info.Types[c.Fun].IsType() {
		if _, m := p.modelForType(p.info.TypeOf(c)); m != nil && m.requiresConstructor() {
			p.fail(c, "use a constructor for %s", m.Name)
		}
	}
	id, ok := c.Fun.(*ast.Ident)
	if !ok {
		return
	}
	if _, ok := p.info.Uses[id].(*types.Builtin); !ok {
		return
	}
	switch id.Name {
	case "new":
		if len(c.Args) == 1 && !p.zeroValid(p.info.TypeOf(c.Args[0])) {
			p.fail(c, "new would create an invalid zero value")
		}
	case "make":
		if len(c.Args) < 2 {
			return
		}
		slice, ok := p.info.TypeOf(c.Args[0]).Underlying().(*types.Slice)
		if ok && !p.zeroValid(slice.Elem()) {
			n, known := integer(p.info, c.Args[1])
			if !known || n != 0 {
				p.fail(c, "make needs constant length 0 for elements with invalid zero values")
			}
		}
	case "clear":
		if len(c.Args) == 1 {
			slice, ok := p.info.TypeOf(c.Args[0]).Underlying().(*types.Slice)
			if ok && !p.zeroValid(slice.Elem()) {
				p.fail(c, "clear would create invalid slice elements")
			}
		}
	}
}

func (p *packageUnit) checkPresence(e ast.Expr, parents map[ast.Node]ast.Node) {
	assign, ok := parents[e].(*ast.AssignStmt)
	if !ok || len(assign.Rhs) != 1 || len(assign.Lhs) != 2 || assign.Tok != token.DEFINE {
		p.fail(e, "this read needs if value, ok := read; ok { ... }")
		return
	}
	branch, ok := parents[assign].(*ast.IfStmt)
	flag, flagOK := assign.Lhs[1].(*ast.Ident)
	value, valueOK := assign.Lhs[0].(*ast.Ident)
	var cond *ast.Ident
	condOK := false
	if ok {
		cond, condOK = branch.Cond.(*ast.Ident)
	}
	if !ok || !flagOK || !valueOK || !condOK {
		p.fail(e, "this read needs if value, ok := read; ok { ... }")
		return
	}
	if branch.Init != assign || p.info.Uses[cond] != p.info.Defs[flag] {
		p.fail(e, "this read needs if value, ok := read; ok { ... }")
		return
	}
	if value.Name == "_" {
		return
	}
	obj := p.info.Defs[value]
	if branch.Else == nil {
		return
	}
	ast.Inspect(branch.Else, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && p.info.Uses[id] == obj {
			p.fail(id, "value is available only in the successful presence branch")
		}
		return true
	})
}

func (p *packageUnit) checkSlice(s *ast.SliceExpr, parents map[ast.Node]ast.Node) {
	t := p.info.TypeOf(s.X)
	if t == nil {
		return
	}
	slice, ok := t.Underlying().(*types.Slice)
	if !ok || p.zeroValid(slice.Elem()) || s.High == nil {
		return
	}
	if p.isLength(s.High, s.X) {
		return
	}
	if n, ok := integer(p.info, s.High); ok && n == 0 {
		return
	}
	// Existing branch tests can prove the bound. No new runtime test is emitted.
	child := ast.Node(s)
	for parent := parents[child]; parent != nil; child, parent = parent, parents[parent] {
		branch, ok := parent.(*ast.IfStmt)
		if !ok || child != branch.Body {
			continue
		}
		if p.bounds(branch.Cond, s.High, s.X) && p.boundUnchanged(branch.Body, s) {
			return
		}
	}
	p.fail(s, "reslice bound must be proven no greater than the current length")
}
func (p *packageUnit) same(a, b ast.Expr) bool {
	x, ok := a.(*ast.Ident)
	y, other := b.(*ast.Ident)
	return ok && other && p.info.ObjectOf(x) == p.info.ObjectOf(y)
}
func (p *packageUnit) bounds(cond, high, slice ast.Expr) bool {
	b, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	if b.Op == token.LAND {
		return p.bounds(b.X, high, slice) || p.bounds(b.Y, high, slice)
	}
	left, right := b.X, b.Y
	if b.Op == token.GEQ || b.Op == token.GTR {
		left, right = right, left
	} else if b.Op != token.LEQ && b.Op != token.LSS {
		return false
	}
	return p.same(left, high) && p.isLength(right, slice)
}

func (p *packageUnit) checkResults(fn *ast.FuncDecl) {
	if fn.Type.Results == nil {
		return
	}
	results := map[types.Object]bool{}
	for _, f := range fn.Type.Results.List {
		for _, name := range f.Names {
			results[p.info.Defs[name]] = false
		}
	}
	if len(results) == 0 {
		return
	}
	p.resultBlock(fn.Body.List, results)
}
func cloneState(state map[types.Object]bool) map[types.Object]bool {
	result := map[types.Object]bool{}
	for k, v := range state {
		result[k] = v
	}
	return result
}
func (p *packageUnit) resultReads(n ast.Node, state map[types.Object]bool) {
	if n == nil {
		return
	}
	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			if initialized, found := state[p.info.Uses[id]]; found && !initialized {
				p.fail(id, "named result %s needs an assignment before use", id.Name)
			}
		}
		return true
	})
}
func (p *packageUnit) resultBlock(list []ast.Stmt, state map[types.Object]bool) bool {
	for _, stmt := range list {
		switch x := stmt.(type) {
		case *ast.AssignStmt:
			for _, e := range x.Rhs {
				p.resultReads(e, state)
			}
			for _, e := range x.Lhs {
				id, ok := e.(*ast.Ident)
				if ok && (x.Tok == token.ASSIGN || x.Tok == token.DEFINE) {
					obj := p.info.ObjectOf(id)
					if _, found := state[obj]; found {
						state[obj] = true
					}
				} else {
					p.resultReads(e, state)
				}
			}
		case *ast.ReturnStmt:
			if len(x.Results) == 0 {
				for obj, valid := range state {
					if !valid {
						p.fail(x, "named result %s needs an assignment before return", obj.Name())
					}
				}
			}
			for _, e := range x.Results {
				p.resultReads(e, state)
			}
			return true
		case *ast.BlockStmt:
			if p.resultBlock(x.List, state) {
				return true
			}
		case *ast.IfStmt:
			if x.Init != nil {
				p.resultBlock([]ast.Stmt{x.Init}, state)
			}
			p.resultReads(x.Cond, state)
			yes, no := cloneState(state), cloneState(state)
			yesReturns := p.resultBlock(x.Body.List, yes)
			noReturns := false
			if x.Else != nil {
				noReturns = p.resultBlock([]ast.Stmt{x.Else}, no)
			}
			for k := range state {
				state[k] = (yes[k] || yesReturns) && (no[k] || noReturns)
			}
			if yesReturns && noReturns {
				return true
			}
		default:
			p.resultReads(stmt, state)
		}
	}
	return false
}

func (p *packageUnit) isLength(expression, slice ast.Expr) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok || !ident(call.Fun, "len") || len(call.Args) != 1 {
		return false
	}
	_, builtin := p.info.Uses[call.Fun.(*ast.Ident)].(*types.Builtin)
	return builtin && p.same(call.Args[0], slice)
}

func firstResult(typ types.Type) types.Type {
	if tuple, ok := typ.(*types.Tuple); ok && tuple.Len() > 0 {
		return tuple.At(0).Type()
	}
	return typ
}

// A bound test stops proving a bound when either operand can change.
func (p *packageUnit) boundUnchanged(body *ast.BlockStmt, slice *ast.SliceExpr) bool {
	unchanged := true
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil || node.Pos() >= slice.Pos() {
			return false
		}
		switch node := node.(type) {
		case *ast.AssignStmt:
			for _, left := range node.Lhs {
				if p.same(left, slice.X) || p.same(left, slice.High) {
					unchanged = false
				}
			}
		case *ast.IncDecStmt:
			if p.same(node.X, slice.High) {
				unchanged = false
			}
		case *ast.CallExpr:
			name, ok := node.Fun.(*ast.Ident)
			if !ok {
				unchanged = false
				break
			}
			_, builtin := p.info.Uses[name].(*types.Builtin)
			if !builtin || name.Name != "len" && name.Name != "cap" {
				unchanged = false
			}
		case *ast.UnaryExpr:
			if node.Op == token.AND && (p.same(node.X, slice.High) || p.same(node.X, slice.X)) {
				unchanged = false
			}
		}
		return unchanged
	})
	return unchanged
}
