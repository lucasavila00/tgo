package compiler

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
)

// zeroValid reports whether Go can make a valid value by zero filling a type.
func (p *packageUnit) zeroValid(t types.Type) bool {
	return p.zero(t, map[types.Type]bool{})
}

// zero checks zero validity and stops recursion through named cycles.
func (p *packageUnit) zero(t types.Type, seen map[types.Type]bool) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if seen[t] {
		return true
	}
	seen[t] = true
	if model := p.modelForType(t); model != nil && model.requiresConstructor() {
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

// constraintZero checks whether all allowed types have a valid zero.
func (p *packageUnit) constraintZero(t types.Type, seen map[types.Type]bool) bool {
	iface, ok := t.Underlying().(*types.Interface)
	if !ok {
		return p.zero(t, seen)
	}
	// One bounded embedded term is enough to limit the intersection.
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		embedded := iface.EmbeddedType(i)
		if union, ok := embedded.(*types.Union); ok {
			if p.unionZeroValid(union) {
				return true
			}
			continue
		}
		if nested, ok := embedded.Underlying().(*types.Interface); ok {
			if p.constraintZero(nested, seen) {
				return true
			}
			continue
		}
		if p.zero(embedded, seen) {
			return true
		}
	}
	return false
}

// unionZeroValid checks every term in one type union.
func (p *packageUnit) unionZeroValid(union *types.Union) bool {
	for i := 0; i < union.Len(); i++ {
		if !p.zero(union.Term(i).Type(), map[types.Type]bool{}) {
			return false
		}
	}
	return true
}

// integer reads a constant integer expression.
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

// ident reports whether an expression is an identifier with the given name.
func ident(e ast.Expr, name string) bool { id, ok := e.(*ast.Ident); return ok && id.Name == name }

// checkRules applies tgo safety rules after Go type checking.
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

// parentNodes maps each AST node to its direct parent.
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

// checkNode sends one source node to its applicable tgo checks.
func (p *packageUnit) checkNode(node ast.Node, parents map[ast.Node]ast.Node) {
	switch node := node.(type) {
	case *ast.ValueSpec:
		p.checkValueSpec(node, parents[node])
	case *ast.CompositeLit:
		p.checkLiteral(node)
	case *ast.CallExpr:
		p.checkCall(node)
	case *ast.SelectorExpr:
		p.checkSelector(node)
	default:
		p.checkCollectionNode(node, parents)
	}
}

// checkValueSpec requires an initializer for each variable declaration.
func (p *packageUnit) checkValueSpec(spec *ast.ValueSpec, parent ast.Node) {
	declaration, ok := parent.(*ast.GenDecl)
	repeatedConstant := ok && declaration.Tok == token.CONST
	if len(spec.Values) == 0 && !repeatedConstant {
		p.fail(spec, "variables need an initializer")
	}
}

// checkCollectionNode checks operations that can expose invalid zero values.
func (p *packageUnit) checkCollectionNode(
	node ast.Node,
	parents map[ast.Node]ast.Node,
) {
	switch node := node.(type) {
	case *ast.SliceExpr:
		p.checkSlice(node, parents)
	case *ast.IndexExpr:
		p.checkMapRead(node, parents)
	case *ast.UnaryExpr:
		invalidReceive := node.Op == token.ARROW &&
			!p.zeroValid(firstResult(p.info.TypeOf(node)))
		if invalidReceive {
			p.checkPresence(node, parents)
		}
	case *ast.TypeAssertExpr:
		invalidAssertion := node.Type != nil &&
			!p.zeroValid(firstResult(p.info.TypeOf(node)))
		if invalidAssertion {
			p.checkPresence(node, parents)
		}
	}
}

// checkSelector blocks direct access to generated model representation.
func (p *packageUnit) checkSelector(selector *ast.SelectorExpr) {
	if selector.Sel.Pos() == token.NoPos {
		return
	}
	typ := p.info.TypeOf(selector.X)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	model := p.modelForType(typ)
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

// checkMapRead requires a presence guard for a map value with an invalid zero.
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

// checkLiteral requires full fields and valid model construction.
func (p *packageUnit) checkLiteral(lit *ast.CompositeLit) {
	t := p.info.TypeOf(lit)
	if t == nil {
		return
	}
	if model := p.modelForType(t); model != nil && model.requiresConstructor() {
		p.fail(lit, "use a constructor for %s", model.Name)
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

// checkElements rejects omitted array or slice elements.
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

// checkCall applies tgo rules to conversions and zero-producing built-ins.
func (p *packageUnit) checkCall(c *ast.CallExpr) {
	if p.checkConversion(c) {
		return
	}

	name, ok := c.Fun.(*ast.Ident)
	if !ok {
		return
	}
	if _, ok := p.info.Uses[name].(*types.Builtin); !ok {
		return
	}

	switch name.Name {
	case "new":
		p.checkNew(c)
	case "make":
		p.checkMake(c)
	case "clear":
		p.checkClear(c)
	}
}

// checkConversion rejects conversions that bypass a model constructor.
func (p *packageUnit) checkConversion(call *ast.CallExpr) bool {
	if !p.info.Types[call.Fun].IsType() {
		return false
	}
	model := p.modelForType(p.info.TypeOf(call))
	if model != nil && model.requiresConstructor() {
		p.fail(call, "use a constructor for %s", model.Name)
	}
	return true
}

// checkNew rejects allocation of a type with an invalid zero value.
func (p *packageUnit) checkNew(call *ast.CallExpr) {
	if len(call.Args) == 1 && !p.zeroValid(p.info.TypeOf(call.Args[0])) {
		p.fail(call, "new would create an invalid zero value")
	}
}

// checkMake permits invalid-zero slice elements only at length zero.
func (p *packageUnit) checkMake(call *ast.CallExpr) {
	if len(call.Args) < 2 {
		return
	}
	slice, ok := p.info.TypeOf(call.Args[0]).Underlying().(*types.Slice)
	if !ok || p.zeroValid(slice.Elem()) {
		return
	}
	length, constant := integer(p.info, call.Args[1])
	if !constant || length != 0 {
		p.fail(call, "make needs constant length 0 for elements with invalid zero values")
	}
}

// checkClear rejects zeroing slice elements that need construction.
func (p *packageUnit) checkClear(call *ast.CallExpr) {
	if len(call.Args) != 1 {
		return
	}
	slice, ok := p.info.TypeOf(call.Args[0]).Underlying().(*types.Slice)
	if ok && !p.zeroValid(slice.Elem()) {
		p.fail(call, "clear would create invalid slice elements")
	}
}

// checkPresence requires a direct presence guard around a risky read.
func (p *packageUnit) checkPresence(e ast.Expr, parents map[ast.Node]ast.Node) {
	guard, ok := p.presenceGuard(e, parents)
	if !ok {
		p.fail(e, "this read needs if value, ok := read; ok { ... }")
		return
	}
	if guard.value.Name == "_" || guard.branch.Else == nil {
		return
	}

	object := p.info.Defs[guard.value]
	ast.Inspect(guard.branch.Else, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && p.info.Uses[id] == object {
			p.fail(id, "value is available only in the successful presence branch")
		}
		return true
	})
}

type presenceGuard struct {
	branch *ast.IfStmt
	value  *ast.Ident
}

// presenceGuard reads the value and success flag from a valid if guard.
func (p *packageUnit) presenceGuard(
	read ast.Expr,
	parents map[ast.Node]ast.Node,
) (presenceGuard, bool) {
	assignment, ok := parents[read].(*ast.AssignStmt)
	if !ok || !isPresenceAssignment(assignment) {
		return presenceGuard{}, false
	}
	branch, ok := parents[assignment].(*ast.IfStmt)
	if !ok || branch.Init != assignment {
		return presenceGuard{}, false
	}
	value, valueOK := assignment.Lhs[0].(*ast.Ident)
	flag, flagOK := assignment.Lhs[1].(*ast.Ident)
	condition, conditionOK := branch.Cond.(*ast.Ident)
	if !valueOK || !flagOK || !conditionOK {
		return presenceGuard{}, false
	}
	if p.info.Uses[condition] != p.info.Defs[flag] {
		return presenceGuard{}, false
	}
	return presenceGuard{branch: branch, value: value}, true
}

// isPresenceAssignment recognizes a two-value short declaration.
func isPresenceAssignment(assignment *ast.AssignStmt) bool {
	return assignment.Tok == token.DEFINE &&
		len(assignment.Rhs) == 1 &&
		len(assignment.Lhs) == 2
}

// checkSlice requires proof before a reslice can expose new elements.
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

// same reports whether two identifiers refer to the same Go object.
func (p *packageUnit) same(a, b ast.Expr) bool {
	x, ok := a.(*ast.Ident)
	y, other := b.(*ast.Ident)
	return ok && other && p.info.ObjectOf(x) == p.info.ObjectOf(y)
}

// bounds recognizes a condition that limits a bound to the current length.
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

// checkResults requires named results to be assigned on every return path.
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

// cloneState copies named-result state for a control-flow branch.
func cloneState(state map[types.Object]bool) map[types.Object]bool {
	result := map[types.Object]bool{}
	for k, v := range state {
		result[k] = v
	}
	return result
}

// resultReads reports reads of named results before assignment.
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

// resultBlock checks named-result state through one statement block.
func (p *packageUnit) resultBlock(list []ast.Stmt, state map[types.Object]bool) bool {
	for _, statement := range list {
		if p.resultStatement(statement, state) {
			return true
		}
	}
	return false
}

// resultStatement updates named-result state for one statement.
func (p *packageUnit) resultStatement(
	statement ast.Stmt,
	state map[types.Object]bool,
) bool {
	switch statement := statement.(type) {
	case *ast.AssignStmt:
		p.resultAssignment(statement, state)
	case *ast.ReturnStmt:
		p.resultReturn(statement, state)
		return true
	case *ast.BlockStmt:
		return p.resultBlock(statement.List, state)
	case *ast.IfStmt:
		return p.resultIf(statement, state)
	default:
		p.resultReads(statement, state)
	}
	return false
}

// resultAssignment marks assigned named results and checks other expressions.
func (p *packageUnit) resultAssignment(
	assignment *ast.AssignStmt,
	state map[types.Object]bool,
) {
	for _, expression := range assignment.Rhs {
		p.resultReads(expression, state)
	}
	for _, expression := range assignment.Lhs {
		name, ok := expression.(*ast.Ident)
		plainAssignment := assignment.Tok == token.ASSIGN || assignment.Tok == token.DEFINE
		if !ok || !plainAssignment {
			p.resultReads(expression, state)
			continue
		}
		object := p.info.ObjectOf(name)
		if _, found := state[object]; found {
			state[object] = true
		}
	}
}

// resultReturn checks reads and bare-return assignment requirements.
func (p *packageUnit) resultReturn(
	statement *ast.ReturnStmt,
	state map[types.Object]bool,
) {
	if len(statement.Results) == 0 {
		for object, initialized := range state {
			if !initialized {
				p.fail(
					statement,
					"named result %s needs an assignment before return",
					object.Name(),
				)
			}
		}
	}
	for _, expression := range statement.Results {
		p.resultReads(expression, state)
	}
}

// resultIf merges named-result state from both branches.
func (p *packageUnit) resultIf(
	statement *ast.IfStmt,
	state map[types.Object]bool,
) bool {
	if statement.Init != nil {
		p.resultStatement(statement.Init, state)
	}
	p.resultReads(statement.Cond, state)

	trueState := cloneState(state)
	falseState := cloneState(state)
	trueReturns := p.resultBlock(statement.Body.List, trueState)
	falseReturns := false
	if statement.Else != nil {
		falseReturns = p.resultStatement(statement.Else, falseState)
	}
	for object := range state {
		truePath := trueState[object] || trueReturns
		falsePath := falseState[object] || falseReturns
		state[object] = truePath && falsePath
	}
	return trueReturns && falseReturns
}

// isLength recognizes len applied to the same slice object.
func (p *packageUnit) isLength(expression, slice ast.Expr) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok || !ident(call.Fun, "len") || len(call.Args) != 1 {
		return false
	}
	_, builtin := p.info.Uses[call.Fun.(*ast.Ident)].(*types.Builtin)
	return builtin && p.same(call.Args[0], slice)
}

// firstResult gets the first value type from a multi-value expression.
func firstResult(typ types.Type) types.Type {
	if tuple, ok := typ.(*types.Tuple); ok && tuple.Len() > 0 {
		return tuple.At(0).Type()
	}
	return typ
}

// A bound test stops proving a bound when either operand can change.
func (p *packageUnit) boundUnchanged(body *ast.BlockStmt, slice *ast.SliceExpr) bool {
	changed := false
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil || node.Pos() >= slice.Pos() {
			return false
		}
		if p.changesBound(node, slice) {
			changed = true
			return false
		}
		return true
	})
	return !changed
}

// changesBound reports whether one node can invalidate a prior bound proof.
func (p *packageUnit) changesBound(node ast.Node, slice *ast.SliceExpr) bool {
	switch node := node.(type) {
	case *ast.AssignStmt:
		return p.assignmentChangesBound(node, slice)
	case *ast.IncDecStmt:
		return p.same(node.X, slice.High)
	case *ast.CallExpr:
		return p.callMayChangeBound(node)
	case *ast.UnaryExpr:
		return node.Op == token.AND &&
			(p.same(node.X, slice.High) || p.same(node.X, slice.X))
	default:
		return false
	}
}

// assignmentChangesBound reports writes to the slice or bound objects.
func (p *packageUnit) assignmentChangesBound(
	assignment *ast.AssignStmt,
	slice *ast.SliceExpr,
) bool {
	for _, left := range assignment.Lhs {
		if p.same(left, slice.X) || p.same(left, slice.High) {
			return true
		}
	}
	return false
}

// callMayChangeBound trusts only len and cap while a bound proof is active.
func (p *packageUnit) callMayChangeBound(call *ast.CallExpr) bool {
	name, ok := call.Fun.(*ast.Ident)
	if !ok {
		return true
	}
	_, builtin := p.info.Uses[name].(*types.Builtin)
	return !builtin || name.Name != "len" && name.Name != "cap"
}
