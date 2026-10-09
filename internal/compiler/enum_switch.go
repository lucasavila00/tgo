package compiler

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/cfg"
)

// checkEnumSwitches checks exhaustive enum tag switches and records permitted accessors.
func (p *packageUnit) checkEnumSwitches(
	root ast.Node,
	parents map[ast.Node]ast.Node,
) (map[*ast.SelectorExpr]bool, map[*ast.SelectorExpr]bool) {
	safe := make(map[*ast.SelectorExpr]bool)
	handled := make(map[*ast.SelectorExpr]bool)
	ast.Inspect(root, func(node ast.Node) bool {
		statement, ok := node.(*ast.SwitchStmt)
		if ok {
			p.checkEnumSwitch(statement, parents, safe, handled)
		}
		return true
	})
	return safe, handled
}

// checkEnumSwitch checks one switch whose tag is a generated TgoTag call.
func (p *packageUnit) checkEnumSwitch(
	statement *ast.SwitchStmt,
	parents map[ast.Node]ast.Node,
	safe map[*ast.SelectorExpr]bool,
	handled map[*ast.SelectorExpr]bool,
) {
	receiver, selector, model := p.enumTagCall(statement.Tag)
	if model == nil {
		return
	}
	safe[selector] = true
	if p.enumTagReceiverUnsafe(receiver, selector, parents) {
		p.fail(selector, "%s.TgoTag needs an unaliased local value receiver", model.Name)
	}
	seen := make(map[int]bool)
	hasSafeDefault := false
	incomingFallthrough := false
	for _, item := range statement.Body.List {
		clause := item.(*ast.CaseClause)
		fallthroughBranch := enumClauseFallthrough(clause)
		if fallthroughBranch != nil {
			p.fail(fallthroughBranch, "TgoTag switch cases cannot fall through")
		}
		if len(clause.List) == 0 {
			hasSafeDefault = p.enumStatementsTerminate(clause.Body, parents)
			incomingFallthrough = fallthroughBranch != nil
			continue
		}
		tags := p.enumCaseTags(clause, len(model.Variants))
		for tag := range tags {
			seen[tag] = true
		}
		p.checkEnumCaseAccessors(
			clause, receiver, model, tags, incomingFallthrough,
			parents, safe, handled,
		)
		incomingFallthrough = fallthroughBranch != nil
	}
	for index, variant := range model.Variants {
		tag := index + 1
		if !seen[tag] {
			p.fail(statement, "switch on %s.TgoTag is missing tag %d (%s)",
				model.Name, tag, variant.Name)
		}
	}
	if !hasSafeDefault {
		p.fail(statement,
			"switch on %s.TgoTag needs a default path that cannot continue",
			model.Name,
		)
	}
}

// enumTagCall resolves a generated tag call and its receiver model.
func (p *packageUnit) enumTagCall(expression ast.Expr) (ast.Expr, *ast.SelectorExpr, *model) {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return nil, nil, nil
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "TgoTag" {
		return nil, nil, nil
	}
	selection := p.info.Selections[selector]
	if selection == nil || !p.enumAccessor(selection, selector.Sel.Name) {
		return nil, nil, nil
	}
	function, ok := selection.Obj().(*types.Func)
	if !ok {
		return nil, nil, nil
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil, nil, nil
	}
	model := p.modelForType(dereference(signature.Recv().Type()))
	if model == nil || len(model.Variants) == 0 {
		return nil, nil, nil
	}
	return selector.X, selector, model
}

func (p *packageUnit) enumCaseTags(clause *ast.CaseClause, variants int) map[int]bool {
	tags := make(map[int]bool)
	for _, expression := range clause.List {
		value := p.info.Types[expression].Value
		if value == nil || value.Kind() != constant.Int {
			p.fail(expression, "tgo enum tag case must be a constant integer")
			continue
		}
		tag64, exact := constant.Int64Val(value)
		if !exact || tag64 < 1 || tag64 > int64(variants) {
			p.fail(expression, "tgo enum tag case is outside the variant range")
			continue
		}
		tags[int(tag64)] = true
	}
	return tags
}

func (p *packageUnit) checkEnumCaseAccessors(
	clause *ast.CaseClause,
	receiver ast.Expr,
	model *model,
	tags map[int]bool,
	incomingFallthrough bool,
	parents map[ast.Node]ast.Node,
	safe map[*ast.SelectorExpr]bool,
	handled map[*ast.SelectorExpr]bool,
) {
	for _, statement := range clause.Body {
		ast.Inspect(statement, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.FuncLit, *ast.GoStmt, *ast.DeferStmt:
				return false
			}
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || !sameEnumReceiver(p.info, receiver, selector.X) {
				return true
			}
			tag := enumVariantTag(model, selector.Sel.Name)
			if tag == 0 {
				return true
			}
			handled[selector] = true
			if p.enumReceiverUnsafe(receiver, selector, parents) {
				p.fail(selector, "%s.%s needs an unaliased local value receiver",
					model.Name, selector.Sel.Name)
				return true
			}
			if enumReceiverChangedBefore(p.info, clause.Body, receiver, selector.Pos()) {
				p.fail(selector, "%s.%s receiver changed after its TgoTag read",
					model.Name, selector.Sel.Name)
				return true
			}
			if incomingFallthrough {
				p.fail(selector, "%s.%s can run after fallthrough from another tag",
					model.Name, selector.Sel.Name)
				return true
			}
			if len(tags) == 1 && tags[tag] {
				safe[selector] = true
				return true
			}
			p.fail(selector, "%s.%s access does not match its TgoTag case %d",
				model.Name, selector.Sel.Name, tag)
			return true
		})
	}
}

func enumVariantTag(model *model, method string) int {
	for index, variant := range model.Variants {
		if method == "Tgo"+variant.Name {
			return index + 1
		}
	}
	return 0
}

func enumClauseFallthrough(clause *ast.CaseClause) *ast.BranchStmt {
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

func (p *packageUnit) enumStatementsTerminate(
	statements []ast.Stmt,
	parents map[ast.Node]ast.Node,
) bool {
	if len(statements) == 0 || p.enumHasEscapingBranch(statements, parents) {
		return false
	}
	sentinel := &ast.ExprStmt{X: ast.NewIdent("__tgo_reached")}
	body := &ast.BlockStmt{List: append(append([]ast.Stmt(nil), statements...), sentinel)}
	graph := cfg.New(body, func(call *ast.CallExpr) bool {
		name, ok := call.Fun.(*ast.Ident)
		if !ok || name.Name != "panic" {
			return true
		}
		_, builtin := p.info.Uses[name].(*types.Builtin)
		return !builtin
	})
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

func (p *packageUnit) enumHasEscapingBranch(
	statements []ast.Stmt,
	parents map[ast.Node]ast.Node,
) bool {
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
				labels[p.info.Defs[label.Label]] = true
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
			escapes = p.enumBranchEscapes(branch, roots, labels, parents)
			return !escapes
		})
		if escapes {
			return true
		}
	}
	return false
}

func (p *packageUnit) enumBranchEscapes(
	branch *ast.BranchStmt,
	roots map[ast.Node]bool,
	labels map[types.Object]bool,
	parents map[ast.Node]ast.Node,
) bool {
	if branch.Tok == token.RETURN {
		return false
	}
	if branch.Label != nil {
		return !labels[p.info.Uses[branch.Label]]
	}
	if branch.Tok != token.BREAK && branch.Tok != token.CONTINUE {
		return true
	}
	if roots[branch] {
		return true
	}
	for node := parents[branch]; node != nil; node = parents[node] {
		if enumBranchTarget(node, branch.Tok) {
			return false
		}
		if roots[node] {
			return true
		}
	}
	return false
}

func enumBranchTarget(node ast.Node, branch token.Token) bool {
	switch node.(type) {
	case *ast.ForStmt, *ast.RangeStmt:
		return true
	case *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return branch == token.BREAK
	default:
		return false
	}
}

func (p *packageUnit) enumReceiverUnsafe(
	receiver ast.Expr,
	accessor *ast.SelectorExpr,
	parents map[ast.Node]ast.Node,
) bool {
	root, _, ok := enumReceiverPath(p.info, receiver)
	variable, variableOK := root.(*types.Var)
	if !ok || !variableOK || variable.IsField() || variable.Parent() == p.typed.Scope() ||
		enumTypeMayBePointer(variable.Type()) {
		return true
	}
	selection := p.info.Selections[accessor]
	if selection != nil && selection.Indirect() {
		return true
	}
	var function ast.Node = accessor
	for parents[function] != nil {
		function = parents[function]
		switch function.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			if variable.Pos() < function.Pos() || variable.Pos() > function.End() {
				return true
			}
			if enumReceiverChangesInCycle(p.info, receiver, accessor, function, parents) ||
				enumReceiverChangesThroughGoto(p.info, receiver, accessor, function, parents) {
				return true
			}
			return enumReceiverCanChangeBefore(p.info, function, receiver, accessor.Pos())
		}
	}
	return true
}

// enumTagReceiverUnsafe rejects receivers that cannot keep one stable tag proof.
func (p *packageUnit) enumTagReceiverUnsafe(
	receiver ast.Expr,
	selector *ast.SelectorExpr,
	parents map[ast.Node]ast.Node,
) bool {
	root, _, ok := enumReceiverPath(p.info, receiver)
	variable, variableOK := root.(*types.Var)
	if !ok || !variableOK || variable.IsField() || variable.Parent() == p.typed.Scope() ||
		enumTypeMayBePointer(variable.Type()) {
		return true
	}
	selection := p.info.Selections[selector]
	if selection != nil && selection.Indirect() {
		return true
	}
	var function ast.Node = selector
	for parents[function] != nil {
		function = parents[function]
		switch function.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			if variable.Pos() < function.Pos() || variable.Pos() > function.End() {
				return true
			}
			return enumReceiverCanChangeBefore(p.info, function, receiver, selector.Pos())
		}
	}
	return true
}

func enumTypeMayBePointer(typ types.Type) bool {
	typ = types.Unalias(typ)
	if _, pointer := typ.(*types.Pointer); pointer {
		return true
	}
	_, parameter := typ.(*types.TypeParam)
	return parameter
}

func enumReceiverChangesThroughGoto(
	info *types.Info,
	receiver ast.Expr,
	accessor *ast.SelectorExpr,
	function ast.Node,
	parents map[ast.Node]ast.Node,
) bool {
	var clause *ast.CaseClause
	for node := parents[accessor]; node != nil && node != function; node = parents[node] {
		if current, ok := node.(*ast.CaseClause); ok {
			clause = current
			break
		}
	}
	if clause == nil || !enumReceiverCanChangeIn(info, clause, receiver) {
		return false
	}
	backward := false
	ast.Inspect(clause, func(node ast.Node) bool {
		branch, ok := node.(*ast.BranchStmt)
		if !ok || branch.Tok != token.GOTO || branch.Label == nil {
			return !backward
		}
		target := info.Uses[branch.Label]
		backward = target != nil && target.Pos() <= accessor.Pos() && branch.Pos() > accessor.Pos()
		return !backward
	})
	return backward
}

func enumReceiverChangesInCycle(
	info *types.Info,
	receiver ast.Expr,
	accessor *ast.SelectorExpr,
	function ast.Node,
	parents map[ast.Node]ast.Node,
) bool {
	for node := parents[accessor]; node != nil && node != function; node = parents[node] {
		if _, clause := node.(*ast.CaseClause); clause {
			return false
		}
		switch node := node.(type) {
		case *ast.ForStmt:
			if enumReceiverCanChangeIn(info, node, receiver) {
				return true
			}
		case *ast.RangeStmt:
			if enumReceiverCanChangeIn(info, node, receiver) {
				return true
			}
		}
	}
	return false
}

func enumReceiverCanChangeIn(info *types.Info, scope ast.Node, receiver ast.Expr) bool {
	root, _, ok := enumReceiverPath(info, receiver)
	if !ok {
		return true
	}
	unsafe := false
	ast.Inspect(scope, func(node ast.Node) bool {
		if node == nil || unsafe {
			return false
		}
		if literal, ok := node.(*ast.FuncLit); ok {
			unsafe = enumCapturesObject(info, literal.Body, root)
			return false
		}
		switch node := node.(type) {
		case *ast.AssignStmt:
			for _, left := range node.Lhs {
				if enumReceiverWrite(info, left, receiver) {
					unsafe = true
					return false
				}
			}
		case *ast.IncDecStmt:
			unsafe = enumReceiverWrite(info, node.X, receiver)
		case *ast.RangeStmt:
			unsafe = enumReceiverWrite(info, node.Key, receiver) ||
				enumReceiverWrite(info, node.Value, receiver)
		case *ast.UnaryExpr:
			unsafe = node.Op == token.AND && enumReceiverWrite(info, node.X, receiver)
		case *ast.CallExpr:
			unsafe = enumPointerMethodCall(info, node, receiver)
		}
		return !unsafe
	})
	return unsafe
}

func enumReceiverCanChangeBefore(
	info *types.Info,
	function ast.Node,
	receiver ast.Expr,
	before token.Pos,
) bool {
	root, _, ok := enumReceiverPath(info, receiver)
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
			unsafe = enumCapturesObject(info, literal.Body, root)
			return false
		}
		unary, ok := node.(*ast.UnaryExpr)
		if ok && unary.Op == token.AND && enumReceiverWrite(info, unary.X, receiver) {
			unsafe = true
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if ok && enumPointerMethodCall(info, call, receiver) {
			unsafe = true
			return false
		}
		return true
	})
	return unsafe
}

func enumCapturesObject(info *types.Info, body *ast.BlockStmt, object types.Object) bool {
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

func enumPointerMethodCall(info *types.Info, call *ast.CallExpr, receiver ast.Expr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !enumReceiverWrite(info, selector.X, receiver) {
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

func enumReceiverChangedBefore(
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
					if enumReceiverWrite(info, left, receiver) {
						changed = true
						return false
					}
				}
			case *ast.IncDecStmt:
				changed = enumReceiverWrite(info, node.X, receiver)
			}
			return !changed
		})
		if changed {
			return true
		}
	}
	return false
}

func enumReceiverWrite(info *types.Info, target, receiver ast.Expr) bool {
	targetRoot, targetPath, targetOK := enumReceiverPath(info, target)
	receiverRoot, receiverFields, receiverOK := enumReceiverPath(info, receiver)
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

func sameEnumReceiver(info *types.Info, left, right ast.Expr) bool {
	leftRoot, leftPath, leftOK := enumReceiverPath(info, left)
	rightRoot, rightPath, rightOK := enumReceiverPath(info, right)
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

func enumReceiverPath(
	info *types.Info,
	expression ast.Expr,
) (types.Object, []types.Object, bool) {
	switch expression := expression.(type) {
	case *ast.Ident:
		object := info.ObjectOf(expression)
		return object, nil, object != nil
	case *ast.SelectorExpr:
		root, path, ok := enumReceiverPath(info, expression.X)
		if !ok {
			return nil, nil, false
		}
		selection := info.Selections[expression]
		if selection == nil || selection.Kind() != types.FieldVal {
			return nil, nil, false
		}
		return root, append(path, selection.Obj()), true
	case *ast.ParenExpr:
		return enumReceiverPath(info, expression.X)
	case *ast.StarExpr:
		return enumReceiverPath(info, expression.X)
	case *ast.UnaryExpr:
		if expression.Op == token.AND {
			return enumReceiverPath(info, expression.X)
		}
	}
	return nil, nil, false
}
