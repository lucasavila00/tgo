package controlflow

import (
	"go/ast"
	"go/token"
)

// Syntax identifies statements inside added expressions and Go control-flow rules.
type Syntax struct {
	RangeSourceOutside func(*ast.RangeStmt) bool
	Statements         func(*ast.CallExpr) ast.Stmt
	Terminates         func(*ast.CallExpr) bool
	Unevaluated        func(*ast.CallExpr) bool
}

// Result reports whether a branch can reach its last statement or its end.
type Result struct {
	Tail bool
	End  bool
}

type flowPoint struct{ successors []*flowPoint }
type jumpTargets struct{ breakTarget, continueTarget, fallthroughTarget *flowPoint }
type branchBuilder struct {
	syntax         Syntax
	labels         map[string]*flowPoint
	jumps          map[string]jumpTargets
	finalStatement ast.Stmt
	finalEntry     *flowPoint
}

// Branch follows all source paths through a branch. Jumps out of the branch end that path.
func Branch(body *ast.BlockStmt, syntax Syntax) Result {
	builder := branchBuilder{syntax: syntax, labels: make(map[string]*flowPoint), jumps: make(map[string]jumpTargets)}
	paths := builder.build(body)
	reachable := reachablePoints(paths.entry)
	return Result{Tail: reachable[paths.finalEntry], End: reachable[paths.end]}
}

// branchPaths is complete before reachability starts. Building needs mutable
// label entries for forward jumps; traversal only reads the completed links.
type branchPaths struct {
	entry, finalEntry, end *flowPoint
}

func (builder *branchBuilder) build(body *ast.BlockStmt) branchPaths {
	list := body.List
	for len(list) > 0 {
		if _, empty := list[len(list)-1].(*ast.EmptyStmt); !empty {
			break
		}
		list = list[:len(list)-1]
	}
	if len(list) > 0 {
		builder.finalStatement = list[len(list)-1]
	}
	end := &flowPoint{}
	entry := builder.statements(body.List, end, jumpTargets{})
	return branchPaths{entry: entry, finalEntry: builder.finalEntry, end: end}
}

func reachablePoints(entry *flowPoint) map[*flowPoint]bool {
	reachable := make(map[*flowPoint]bool)
	pending := []*flowPoint{entry}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if current == nil || reachable[current] {
			continue
		}
		reachable[current] = true
		pending = append(pending, current.successors...)
	}
	return reachable
}

func link(continuation ...*flowPoint) *flowPoint { return &flowPoint{successors: continuation} }
func (builder *branchBuilder) label(name string) *flowPoint {
	if builder.labels[name] == nil {
		builder.labels[name] = &flowPoint{}
	}
	return builder.labels[name]
}
func (builder *branchBuilder) statements(list []ast.Stmt, continuation *flowPoint, targets jumpTargets) *flowPoint {
	for i := len(list) - 1; i >= 0; i-- {
		continuation = builder.statement(list[i], continuation, targets, "")
	}
	return continuation
}
func (builder *branchBuilder) statement(
	statement ast.Stmt,
	continuation *flowPoint,
	targets jumpTargets,
	label string,
) *flowPoint {
	entry := builder.statementInner(statement, continuation, targets, label)
	if statement == builder.finalStatement {
		if _, labelled := statement.(*ast.LabeledStmt); labelled {
			builder.finalEntry = entry
			return entry
		}
		builder.finalEntry = link(entry)
		return builder.finalEntry
	}
	return entry
}
func (builder *branchBuilder) statementInner(
	statement ast.Stmt,
	continuation *flowPoint,
	targets jumpTargets,
	label string,
) *flowPoint {
	if statement == nil {
		return continuation
	}
	switch statement := statement.(type) {
	case *ast.BlockStmt:
		return builder.statements(statement.List, continuation, targets)
	case *ast.LabeledStmt:
		entry := builder.label(statement.Label.Name)
		entry.successors = []*flowPoint{builder.statement(statement.Stmt, continuation, targets, statement.Label.Name)}
		return entry
	case *ast.BranchStmt:
		return builder.branchStatement(statement, continuation, targets)
	case *ast.ReturnStmt:
		return builder.expressions(statement.Results, link(), targets)
	case *ast.ExprStmt:
		return builder.expression(statement.X, continuation, targets)
	case *ast.AssignStmt:
		operands := append([]ast.Expr{}, statement.Lhs...)
		operands = append(operands, statement.Rhs...)
		return builder.expressions(operands, continuation, targets)
	case *ast.DeclStmt:
		return builder.declaration(statement, continuation, targets)
	case *ast.SendStmt:
		return builder.expressions([]ast.Expr{statement.Chan, statement.Value}, continuation, targets)
	case *ast.IncDecStmt:
		return builder.expression(statement.X, continuation, targets)
	case *ast.GoStmt:
		return builder.callOperands(statement.Call, continuation, targets)
	case *ast.DeferStmt:
		return builder.callOperands(statement.Call, continuation, targets)
	case *ast.IfStmt:
		choice := link(
			builder.statement(statement.Body, continuation, targets, ""),
			builder.statement(statement.Else, continuation, targets, ""),
		)
		return builder.statement(statement.Init, builder.expression(statement.Cond, choice, targets), targets, "")
	case *ast.ForStmt:
		return builder.forStatement(statement, continuation, targets, label)
	case *ast.RangeStmt:
		return builder.rangeStatement(statement, continuation, targets, label)
	case *ast.SwitchStmt:
		return builder.switchStatement(statement, continuation, targets, label)
	case *ast.TypeSwitchStmt:
		return builder.typeSwitch(statement, continuation, targets, label)
	case *ast.SelectStmt:
		return builder.selectStatement(statement, continuation, targets, label)
	}
	return continuation
}

func (builder *branchBuilder) declaration(
	statement *ast.DeclStmt,
	continuation *flowPoint,
	targets jumpTargets,
) *flowPoint {
	if decl, ok := statement.Decl.(*ast.GenDecl); ok {
		for i := len(decl.Specs) - 1; i >= 0; i-- {
			if spec, ok := decl.Specs[i].(*ast.ValueSpec); ok {
				continuation = builder.expressions(spec.Values, continuation, targets)
			}
		}
	}
	return continuation
}

func (builder *branchBuilder) branchStatement(
	statement *ast.BranchStmt,
	continuation *flowPoint,
	targets jumpTargets,
) *flowPoint {
	switch statement.Tok {
	case token.GOTO:
		return builder.label(statement.Label.Name)
	case token.BREAK:
		if statement.Label != nil {
			return link(builder.jumps[statement.Label.Name].breakTarget)
		}
		return link(targets.breakTarget)
	case token.CONTINUE:
		if statement.Label != nil {
			return link(builder.jumps[statement.Label.Name].continueTarget)
		}
		return link(targets.continueTarget)
	case token.FALLTHROUGH:
		return link(targets.fallthroughTarget)
	}
	return continuation
}

func (builder *branchBuilder) expression(expr ast.Expr, continuation *flowPoint, targets jumpTargets) *flowPoint {
	if expr == nil {
		return continuation
	}
	switch expression := expr.(type) {
	case *ast.CallExpr:
		return builder.callExpression(expression, continuation, targets)
	case *ast.BinaryExpr:
		right := builder.expression(expression.Y, continuation, targets)
		if expression.Op == token.LAND || expression.Op == token.LOR {
			right = link(continuation, right)
		}
		return builder.expression(expression.X, right, targets)
	case *ast.UnaryExpr:
		return builder.expression(expression.X, continuation, targets)
	case *ast.ParenExpr:
		return builder.expression(expression.X, continuation, targets)
	case *ast.SelectorExpr:
		return builder.expression(expression.X, continuation, targets)
	case *ast.IndexExpr:
		return builder.expressions([]ast.Expr{expression.X, expression.Index}, continuation, targets)
	case *ast.IndexListExpr:
		return builder.expression(expression.X, continuation, targets)
	case *ast.SliceExpr:
		operands := []ast.Expr{expression.X, expression.Low, expression.High, expression.Max}
		return builder.expressions(operands, continuation, targets)
	case *ast.StarExpr:
		return builder.expression(expression.X, continuation, targets)
	case *ast.TypeAssertExpr:
		return builder.expression(expression.X, continuation, targets)
	case *ast.CompositeLit:
		return builder.expressions(expression.Elts, continuation, targets)
	case *ast.KeyValueExpr:
		return builder.expressions([]ast.Expr{expression.Key, expression.Value}, continuation, targets)
	}
	return continuation
}
func (builder *branchBuilder) callExpression(
	expression *ast.CallExpr,
	continuation *flowPoint,
	targets jumpTargets,
) *flowPoint {
	if builder.syntax.Unevaluated != nil && builder.syntax.Unevaluated(expression) {
		return continuation
	}
	if builder.syntax.Statements != nil {
		if body := builder.syntax.Statements(expression); body != nil {
			return builder.statement(body, continuation, targets, "")
		}
	}
	if builder.syntax.Terminates != nil && builder.syntax.Terminates(expression) {
		continuation = link()
	}
	return builder.callOperands(expression, continuation, targets)
}

func (builder *branchBuilder) expressions(exprs []ast.Expr, continuation *flowPoint, targets jumpTargets) *flowPoint {
	for i := len(exprs) - 1; i >= 0; i-- {
		continuation = builder.expression(exprs[i], continuation, targets)
	}
	return continuation
}
func (builder *branchBuilder) callOperands(
	call *ast.CallExpr,
	continuation *flowPoint,
	targets jumpTargets,
) *flowPoint {
	argumentsEntry := builder.expressions(call.Args, continuation, targets)
	return builder.expression(call.Fun, argumentsEntry, targets)
}
func (builder *branchBuilder) forStatement(
	statement *ast.ForStmt,
	continuation *flowPoint,
	targets jumpTargets,
	label string,
) *flowPoint {
	cond := &flowPoint{}
	post := &flowPoint{}
	bodyTargets := jumpTargets{
		breakTarget: continuation, continueTarget: post, fallthroughTarget: targets.fallthroughTarget,
	}
	if label != "" {
		builder.jumps[label] = bodyTargets
		defer delete(builder.jumps, label)
	}
	post.successors = []*flowPoint{builder.statement(statement.Post, cond, bodyTargets, "")}
	body := builder.statement(statement.Body, post, bodyTargets, "")
	choice := body
	if statement.Cond != nil {
		choice = link(body, continuation)
	}
	cond.successors = []*flowPoint{builder.expression(statement.Cond, choice, bodyTargets)}
	return builder.statement(statement.Init, cond, bodyTargets, "")
}
func (builder *branchBuilder) rangeStatement(
	statement *ast.RangeStmt,
	continuation *flowPoint,
	targets jumpTargets,
	label string,
) *flowPoint {
	step := &flowPoint{}
	bodyTargets := jumpTargets{
		breakTarget: continuation, continueTarget: step, fallthroughTarget: targets.fallthroughTarget,
	}
	if label != "" {
		builder.jumps[label] = bodyTargets
		defer delete(builder.jumps, label)
	}
	bodyEntry := builder.statement(statement.Body, step, bodyTargets, "")
	assignmentEntry := builder.expressions([]ast.Expr{statement.Key, statement.Value}, bodyEntry, bodyTargets)
	step.successors = []*flowPoint{assignmentEntry, continuation}
	if builder.syntax.RangeSourceOutside != nil && builder.syntax.RangeSourceOutside(statement) {
		bodyTargets = targets
	}
	return builder.expression(statement.X, step, bodyTargets)
}
func (builder *branchBuilder) switchStatement(
	statement *ast.SwitchStmt,
	continuation *flowPoint,
	targets jumpTargets,
	label string,
) *flowPoint {
	bodyTargets := jumpTargets{breakTarget: continuation, continueTarget: targets.continueTarget}
	if label != "" {
		builder.jumps[label] = bodyTargets
		defer delete(builder.jumps, label)
	}
	bodies := make([]*flowPoint, len(statement.Body.List))
	fallthroughEntry := continuation
	defaultBody := continuation
	for i := len(bodies) - 1; i >= 0; i-- {
		clause := statement.Body.List[i].(*ast.CaseClause)
		bodyTargets.fallthroughTarget = fallthroughEntry
		bodies[i] = builder.statements(clause.Body, continuation, bodyTargets)
		fallthroughEntry = bodies[i]
		if clause.List == nil {
			defaultBody = bodies[i]
		}
	}
	choice := defaultBody
	for i := len(bodies) - 1; i >= 0; i-- {
		clause := statement.Body.List[i].(*ast.CaseClause)
		for j := len(clause.List) - 1; j >= 0; j-- {
			choice = builder.expression(clause.List[j], link(bodies[i], choice), bodyTargets)
		}
	}
	return builder.statement(statement.Init, builder.expression(statement.Tag, choice, bodyTargets), bodyTargets, "")
}
func (builder *branchBuilder) typeSwitch(
	statement *ast.TypeSwitchStmt,
	continuation *flowPoint,
	targets jumpTargets,
	label string,
) *flowPoint {
	bodyTargets := jumpTargets{breakTarget: continuation, continueTarget: targets.continueTarget}
	if label != "" {
		builder.jumps[label] = bodyTargets
		defer delete(builder.jumps, label)
	}
	choice := &flowPoint{}
	hasDefault := false
	for _, stmt := range statement.Body.List {
		clause := stmt.(*ast.CaseClause)
		hasDefault = hasDefault || clause.List == nil
		choice.successors = append(choice.successors, builder.statements(clause.Body, continuation, bodyTargets))
	}
	if !hasDefault {
		choice.successors = append(choice.successors, continuation)
	}
	return builder.statement(statement.Init, builder.statement(statement.Assign, choice, bodyTargets, ""), bodyTargets, "")
}
func (builder *branchBuilder) selectStatement(
	statement *ast.SelectStmt,
	continuation *flowPoint,
	targets jumpTargets,
	label string,
) *flowPoint {
	bodyTargets := jumpTargets{breakTarget: continuation, continueTarget: targets.continueTarget}
	if label != "" {
		builder.jumps[label] = bodyTargets
		defer delete(builder.jumps, label)
	}
	choice := &flowPoint{}
	var operands []ast.Expr
	for _, stmt := range statement.Body.List {
		clause := stmt.(*ast.CommClause)
		body := builder.statements(clause.Body, continuation, bodyTargets)
		switch comm := clause.Comm.(type) {
		case *ast.SendStmt:
			operands = append(operands, comm.Chan, comm.Value)
		case *ast.ExprStmt:
			operands = append(operands, ast.Unparen(comm.X).(*ast.UnaryExpr).X)
		case *ast.AssignStmt:
			operands = append(operands, ast.Unparen(comm.Rhs[0]).(*ast.UnaryExpr).X)
			body = builder.expressions(comm.Lhs, body, bodyTargets)
		}
		choice.successors = append(choice.successors, body)
	}
	return builder.expressions(operands, choice, bodyTargets)
}
