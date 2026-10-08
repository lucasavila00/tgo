package tgolint

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/cfg"
)

func (c *checker) checkTagSwitch(statement *ast.SwitchStmt) {
	receiver, selector, model := c.tagCall(statement.Tag)
	if model == nil {
		return
	}
	c.safe[selector] = true
	seen := make(map[int]bool)
	hasSafeDefault := false
	incomingFallthrough := false
	for _, item := range statement.Body.List {
		clause := item.(*ast.CaseClause)
		fallthroughBranch := clauseFallthrough(clause)
		if fallthroughBranch != nil {
			c.pass.Reportf(fallthroughBranch.Pos(), "TgoTag switch cases cannot fall through")
		}
		if len(clause.List) == 0 {
			hasSafeDefault = c.defaultStops(clause)
			incomingFallthrough = fallthroughBranch != nil
			continue
		}
		tags := c.caseTags(clause, len(model.Variants))
		for tag := range tags {
			seen[tag] = true
		}
		c.checkCaseAccessors(clause, receiver, model, tags, incomingFallthrough)
		incomingFallthrough = fallthroughBranch != nil
	}
	for index, variant := range model.Variants {
		tag := index + 1
		if !seen[tag] {
			c.pass.Reportf(statement.Switch,
				"switch on %s.TgoTag is missing tag %d (%s)", model.Name, tag, variant)
		}
	}
	if !hasSafeDefault {
		c.pass.Reportf(statement.Switch,
			"switch on %s.TgoTag needs a default path that cannot continue", model.Name)
	}
}

func clauseFallthrough(clause *ast.CaseClause) *ast.BranchStmt {
	for index := len(clause.Body) - 1; index >= 0; index-- {
		statement := clause.Body[index]
		if _, empty := statement.(*ast.EmptyStmt); empty {
			continue
		}
		for {
			label, ok := statement.(*ast.LabeledStmt)
			if !ok {
				break
			}
			statement = label.Stmt
		}
		branch, ok := statement.(*ast.BranchStmt)
		if ok && branch.Tok == token.FALLTHROUGH {
			return branch
		}
		return nil
	}
	return nil
}

func (c *checker) defaultStops(clause *ast.CaseClause) bool {
	return c.statementsTerminate(clause.Body)
}

func (c *checker) statementsTerminate(statements []ast.Stmt) bool {
	if len(statements) == 0 || c.hasEscapingBranch(statements) {
		return false
	}
	sentinel := &ast.ExprStmt{X: ast.NewIdent("__tgolint_reached")}
	body := &ast.BlockStmt{List: append(append([]ast.Stmt(nil), statements...), sentinel)}
	graph := cfg.New(body, c.callMayReturn)
	for _, block := range graph.Blocks {
		if !block.Live {
			continue
		}
		for _, node := range block.Nodes {
			if node == sentinel {
				return false
			}
		}
	}
	return true
}

func (c *checker) hasEscapingBranch(statements []ast.Stmt) bool {
	roots := make(map[ast.Node]bool, len(statements))
	labels := make(map[types.Object]bool)
	for _, statement := range statements {
		roots[statement] = true
		ast.Inspect(statement, func(node ast.Node) bool {
			if _, nested := node.(*ast.FuncLit); nested {
				return false
			}
			label, ok := node.(*ast.LabeledStmt)
			if ok {
				labels[c.pass.TypesInfo.Defs[label.Label]] = true
			}
			return true
		})
	}
	for _, statement := range statements {
		escapes := false
		ast.Inspect(statement, func(node ast.Node) bool {
			if escapes {
				return false
			}
			if _, nested := node.(*ast.FuncLit); nested {
				return false
			}
			branch, ok := node.(*ast.BranchStmt)
			if !ok {
				return true
			}
			escapes = c.branchEscapes(branch, roots, labels)
			return !escapes
		})
		if escapes {
			return true
		}
	}
	return false
}

func (c *checker) branchEscapes(
	branch *ast.BranchStmt,
	roots map[ast.Node]bool,
	labels map[types.Object]bool,
) bool {
	if branch.Label != nil {
		return !labels[c.pass.TypesInfo.Uses[branch.Label]]
	}
	if branch.Tok != token.BREAK && branch.Tok != token.CONTINUE {
		return true
	}
	for node := c.parents[branch]; node != nil; node = c.parents[node] {
		if branchTarget(node, branch.Tok) {
			return false
		}
		if roots[node] {
			return true
		}
	}
	return true
}

func branchTarget(node ast.Node, branch token.Token) bool {
	switch node.(type) {
	case *ast.ForStmt, *ast.RangeStmt:
		return true
	case *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return branch == token.BREAK
	default:
		return false
	}
}

func (c *checker) callMayReturn(call *ast.CallExpr) bool {
	name, ok := call.Fun.(*ast.Ident)
	if !ok || name.Name != "panic" {
		return true
	}
	_, builtin := c.pass.TypesInfo.Uses[name].(*types.Builtin)
	return !builtin
}

func (c *checker) tagCall(expression ast.Expr) (ast.Expr, *ast.SelectorExpr, *modelFact) {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return nil, nil, nil
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "TgoTag" {
		return nil, nil, nil
	}
	model := c.modelForSelector(selector)
	if model == nil || model.Kind != enumKind {
		return nil, nil, nil
	}
	return selector.X, selector, model
}

func (c *checker) caseTags(clause *ast.CaseClause, variants int) map[int]bool {
	tags := make(map[int]bool)
	for _, expression := range clause.List {
		value := c.pass.TypesInfo.Types[expression].Value
		if value == nil || value.Kind() != constant.Int {
			c.pass.Reportf(expression.Pos(), "tgo enum tag case must be a constant integer")
			continue
		}
		tag64, exact := constant.Int64Val(value)
		if !exact || tag64 < 1 || tag64 > int64(variants) {
			c.pass.Reportf(expression.Pos(), "tgo enum tag case is outside the variant range")
			continue
		}
		tags[int(tag64)] = true
	}
	return tags
}

func (c *checker) checkCaseAccessors(
	clause *ast.CaseClause,
	receiver ast.Expr,
	model *modelFact,
	tags map[int]bool,
	incomingFallthrough bool,
) {
	for _, statement := range clause.Body {
		ast.Inspect(statement, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.FuncLit, *ast.GoStmt, *ast.DeferStmt:
				return false
			}
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || !sameReceiver(c.pass.TypesInfo, receiver, selector.X) {
				return true
			}
			tag := variantTag(model, selector.Sel.Name)
			if tag == 0 {
				return true
			}
			c.handled[selector] = true
			if c.receiverUnsafe(receiver, selector) {
				c.pass.Reportf(selector.Pos(),
					"%s.%s needs an unaliased local value receiver",
					model.Name, selector.Sel.Name)
				return true
			}
			if receiverChangedBefore(c.pass.TypesInfo, clause.Body, receiver, selector.Pos()) {
				c.pass.Reportf(selector.Pos(),
					"%s.%s receiver changed after its TgoTag read",
					model.Name, selector.Sel.Name)
				return true
			}
			if incomingFallthrough {
				c.pass.Reportf(selector.Pos(),
					"%s.%s can run after fallthrough from another tag",
					model.Name, selector.Sel.Name)
				return true
			}
			if len(tags) == 1 && tags[tag] {
				c.safe[selector] = true
				return true
			}
			c.pass.Reportf(selector.Pos(),
				"%s.%s access does not match its TgoTag case %d",
				model.Name, selector.Sel.Name, tag)
			return true
		})
	}
}

func (c *checker) receiverUnsafe(receiver ast.Expr, accessor *ast.SelectorExpr) bool {
	root, _, ok := receiverPath(c.pass.TypesInfo, receiver)
	variable, variableOK := root.(*types.Var)
	if !ok || !variableOK || variable.IsField() ||
		variable.Parent() == c.pass.Pkg.Scope() {
		return true
	}
	if typeMayBePointer(variable.Type()) {
		return true
	}
	selection := c.pass.TypesInfo.Selections[accessor]
	if selection != nil && selection.Indirect() {
		return true
	}
	var function ast.Node = accessor
	for c.parents[function] != nil {
		function = c.parents[function]
		switch function.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			if variable.Pos() < function.Pos() || variable.Pos() > function.End() {
				return true
			}
			if c.receiverChangesInCycle(receiver, accessor, function) {
				return true
			}
			if c.receiverChangesThroughGoto(receiver, accessor, function) {
				return true
			}
			return receiverCanChangeBefore(
				c.pass.TypesInfo,
				function,
				receiver,
				accessor.Pos(),
			)
		}
	}
	return true
}

// receiverChangesThroughGoto finds a write on a backward path to an accessor.
// A repeated payload read needs a new tag read after each receiver change.
func (c *checker) receiverChangesThroughGoto(
	receiver ast.Expr,
	accessor *ast.SelectorExpr,
	function ast.Node,
) bool {
	var clause *ast.CaseClause
	for node := c.parents[accessor]; node != nil && node != function; node = c.parents[node] {
		if current, ok := node.(*ast.CaseClause); ok {
			clause = current
			break
		}
	}
	if clause == nil || !receiverCanChangeIn(c.pass.TypesInfo, clause, receiver) {
		return false
	}
	backward := false
	ast.Inspect(clause, func(node ast.Node) bool {
		branch, ok := node.(*ast.BranchStmt)
		if !ok || branch.Tok != token.GOTO || branch.Label == nil {
			return !backward
		}
		target := c.pass.TypesInfo.Uses[branch.Label]
		backward = target != nil && target.Pos() <= accessor.Pos() &&
			branch.Pos() > accessor.Pos()
		return !backward
	})
	return backward
}

func typeMayBePointer(typ types.Type) bool {
	typ = types.Unalias(typ)
	if _, pointer := typ.(*types.Pointer); pointer {
		return true
	}
	parameter, ok := typ.(*types.TypeParam)
	if !ok {
		return false
	}
	terms, supported := simpleTerms(parameter.Constraint())
	if !supported {
		return true
	}
	for _, term := range terms {
		if typeMayBePointer(term.Type()) {
			return true
		}
	}
	return false
}

// receiverChangesInCycle finds a write that can run before a later iteration.
// The tag proof cannot survive a loop that can change or alias the receiver.
func (c *checker) receiverChangesInCycle(
	receiver ast.Expr,
	accessor *ast.SelectorExpr,
	function ast.Node,
) bool {
	for node := c.parents[accessor]; node != nil && node != function; node = c.parents[node] {
		switch node := node.(type) {
		case *ast.ForStmt:
			if receiverCanChangeIn(c.pass.TypesInfo, node, receiver) {
				return true
			}
		case *ast.RangeStmt:
			if receiverCanChangeIn(c.pass.TypesInfo, node, receiver) {
				return true
			}
		}
	}
	return false
}

func receiverCanChangeIn(info *types.Info, scope ast.Node, receiver ast.Expr) bool {
	root, _, ok := receiverPath(info, receiver)
	if !ok {
		return true
	}
	unsafe := false
	ast.Inspect(scope, func(node ast.Node) bool {
		if node == nil || unsafe {
			return false
		}
		if literal, ok := node.(*ast.FuncLit); ok {
			unsafe = capturesObject(info, literal.Body, root)
			return false
		}
		switch node := node.(type) {
		case *ast.AssignStmt:
			for _, left := range node.Lhs {
				if receiverWrite(info, left, receiver) {
					unsafe = true
					return false
				}
			}
		case *ast.IncDecStmt:
			unsafe = receiverWrite(info, node.X, receiver)
		case *ast.UnaryExpr:
			unsafe = node.Op == token.AND && receiverWrite(info, node.X, receiver)
		case *ast.CallExpr:
			unsafe = pointerMethodCall(info, node, receiver)
		}
		return !unsafe
	})
	return unsafe
}

// receiverCanChangeBefore reports paths that can mutate a receiver through an alias.
// It includes explicit addresses, captures, and pointer receiver calls.
func receiverCanChangeBefore(
	info *types.Info,
	function ast.Node,
	receiver ast.Expr,
	before token.Pos,
) bool {
	root, _, ok := receiverPath(info, receiver)
	if !ok {
		return true
	}
	unsafe := false
	ast.Inspect(function, func(node ast.Node) bool {
		if node == nil || node.Pos() >= before || unsafe {
			return false
		}
		literal, nested := node.(*ast.FuncLit)
		if nested && literal != function {
			unsafe = capturesObject(info, literal.Body, root)
			return false
		}
		unary, ok := node.(*ast.UnaryExpr)
		if ok && unary.Op == token.AND && receiverWrite(info, unary.X, receiver) {
			unsafe = true
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if ok && pointerMethodCall(info, call, receiver) {
			unsafe = true
			return false
		}
		return true
	})
	return unsafe
}

func capturesObject(info *types.Info, body *ast.BlockStmt, object types.Object) bool {
	captured := false
	ast.Inspect(body, func(node ast.Node) bool {
		if captured {
			return false
		}
		name, ok := node.(*ast.Ident)
		if ok && info.Uses[name] == object {
			captured = true
		}
		return !captured
	})
	return captured
}

func pointerMethodCall(info *types.Info, call *ast.CallExpr, receiver ast.Expr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !receiverWrite(info, selector.X, receiver) {
		return false
	}
	selection := info.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal {
		return false
	}
	function, ok := selection.Obj().(*types.Func)
	if !ok {
		return false
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	_, pointer := types.Unalias(signature.Recv().Type()).(*types.Pointer)
	return pointer
}

func receiverChangedBefore(
	info *types.Info,
	statements []ast.Stmt,
	receiver ast.Expr,
	before token.Pos,
) bool {
	changed := false
	for _, statement := range statements {
		ast.Inspect(statement, func(node ast.Node) bool {
			if node == nil || node.Pos() >= before || changed {
				return false
			}
			switch node.(type) {
			case *ast.FuncLit, *ast.GoStmt, *ast.DeferStmt:
				return false
			}
			switch node := node.(type) {
			case *ast.AssignStmt:
				for _, left := range node.Lhs {
					if receiverWrite(info, left, receiver) {
						changed = true
						return false
					}
				}
			case *ast.IncDecStmt:
				changed = receiverWrite(info, node.X, receiver)
			}
			return !changed
		})
		if changed {
			return true
		}
	}
	return false
}

func receiverWrite(info *types.Info, target, receiver ast.Expr) bool {
	targetRoot, targetPath, targetOK := receiverPath(info, target)
	receiverRoot, receiverFields, receiverOK := receiverPath(info, receiver)
	if !targetOK || !receiverOK || targetRoot != receiverRoot ||
		len(targetPath) > len(receiverFields) {
		return false
	}
	for index := range targetPath {
		if targetPath[index] != receiverFields[index] {
			return false
		}
	}
	return true
}

func variantTag(model *modelFact, method string) int {
	for index, variant := range model.Variants {
		if method == "Tgo"+variant {
			return index + 1
		}
	}
	return 0
}

func sameReceiver(info *types.Info, left, right ast.Expr) bool {
	leftRoot, leftPath, leftOK := receiverPath(info, left)
	rightRoot, rightPath, rightOK := receiverPath(info, right)
	if !leftOK || !rightOK || leftRoot != rightRoot || len(leftPath) != len(rightPath) {
		return false
	}
	for index := range leftPath {
		if leftPath[index] != rightPath[index] {
			return false
		}
	}
	return true
}

func receiverPath(info *types.Info, expression ast.Expr) (types.Object, []types.Object, bool) {
	switch expression := expression.(type) {
	case *ast.Ident:
		object := info.ObjectOf(expression)
		return object, nil, object != nil
	case *ast.SelectorExpr:
		root, path, ok := receiverPath(info, expression.X)
		if !ok {
			return nil, nil, false
		}
		selection := info.Selections[expression]
		if selection == nil || selection.Kind() != types.FieldVal {
			return nil, nil, false
		}
		return root, append(path, selection.Obj()), true
	case *ast.ParenExpr:
		return receiverPath(info, expression.X)
	case *ast.StarExpr:
		return receiverPath(info, expression.X)
	case *ast.UnaryExpr:
		if expression.Op == token.AND {
			return receiverPath(info, expression.X)
		}
	}
	return nil, nil, false
}

func (c *checker) checkRepresentationAccess(selector *ast.SelectorExpr) {
	if c.safe[selector] || c.handled[selector] {
		return
	}
	model := c.modelForSelector(selector)
	if model == nil {
		return
	}
	if model.Mixed {
		c.pass.Reportf(selector.Pos(),
			"Tgo access through a type parameter cannot mix tgo models")
		return
	}
	if privateRepresentation(model, selector.Sel.Name) {
		c.pass.Reportf(selector.Pos(),
			"%s.%s is private tgo representation", model.Name, selector.Sel.Name)
		return
	}
	if model.Kind != enumKind {
		return
	}
	if selector.Sel.Name == "TgoTag" {
		c.pass.Reportf(selector.Pos(),
			"%s.TgoTag must be the tag of an exhaustive switch", model.Name)
		return
	}
	if variantTag(model, selector.Sel.Name) != 0 {
		c.pass.Reportf(selector.Pos(),
			"%s.%s requires the matching case of an exhaustive TgoTag switch",
			model.Name, selector.Sel.Name)
	}
}

func privateRepresentation(model *modelFact, name string) bool {
	if model.Kind == checkedKind {
		return name == "value"
	}
	if name == "tgoTag" {
		return true
	}
	for _, variant := range model.Variants {
		if name == "tgo"+variant {
			return true
		}
	}
	return false
}
