// Package types checks Go syntax with TGo conditional values and error propagation.
package types

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/controlflow"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// TGoOperation records the meaning of a frontend marker call.
type TGoOperation struct {
	Ranges        []*ast.RangeStmt
	Comprehension bool
	Output        string
	Element       bool
	Conditional   bool
	ShortReturn   bool
	ZeroSuccess   bool
	Wrap          bool
	CallName      string
	Position      token.Position
}

func (check *Checker) tgoOperation(call *ast.CallExpr) (TGoOperation, bool) {
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return TGoOperation{}, false
	}
	operation, found := check.conf.TGo[id.Name]
	return operation, found
}

// expressionType passes context through operators whose result has that type.
func (check *Checker) expressionType(e ast.Expr, typ Type) {
	if e == nil || typ == nil {
		return
	}
	if check.expressionTypes == nil {
		check.expressionTypes = make(map[ast.Expr]Type)
	}
	check.expressionTypes[e] = typ
	switch e := e.(type) {
	case *ast.ParenExpr:
		check.expressionType(e.X, typ)
	case *ast.UnaryExpr:
		if accepts := unaryOpPredicates[e.Op]; accepts != nil && accepts(typ) {
			check.expressionType(e.X, typ)
		}
	case *ast.BinaryExpr:
		if accepts := binaryOpPredicates[e.Op]; (accepts != nil && accepts(typ)) || (isShift(e.Op) && allInteger(typ)) {
			check.expressionType(e.X, typ)
			if !isShift(e.Op) {
				check.expressionType(e.Y, typ)
			}
		}
	}
}

// builtinArguments supplies declared parameter types to added expressions.
func (check *Checker) builtinArguments(call *ast.CallExpr, id builtinId) []*operand {
	previous := check.sliceSourceContext
	check.sliceSourceContext = nil
	defer func() { check.sliceSourceContext = previous }()
	if id == _Min || id == _Max || id == _Complex {
		return check.numericBuiltinArguments(call)
	}
	if id == _Panic && len(call.Args) == 1 {
		check.expressionType(call.Args[0], &emptyInterface)
		return check.exprList(call.Args)
	}
	if (id != _Append && id != _Delete && id != _Copy) || len(call.Args) < 2 {
		return check.exprList(call.Args)
	}
	args := make([]*operand, len(call.Args))
	for i, expression := range call.Args {
		args[i] = new(operand)
		check.expr(nil, args[i], expression)
		if i == 0 && args[i].isValid() {
			var typ Type
			switch {
			case id == _Delete && !hasDots(call):
				typ = deleteKeyType(args[i].typ())
			case id == _Append && !hasDots(call):
				typ, _ = sliceElem(args[i])
			case id == _Copy || id == _Append && hasDots(call):
				if elem, _ := sliceElem(args[i]); elem != nil {
					typ = NewSlice(elem)
					if Identical(elem, universeByte) {
						check.sliceSourceContext = typ
					}
				}
			}
			for _, value := range call.Args[1:] {
				check.expressionType(value, typ)
			}
		}
	}
	return args
}

// numericBuiltinArguments lets another argument supply a terminating if's type.
func (check *Checker) numericBuiltinArguments(call *ast.CallExpr) []*operand {
	conditional := false
	for _, expression := range call.Args {
		if marker, ok := ast.Unparen(expression).(*ast.CallExpr); ok {
			if operation, found := check.tgoOperation(marker); found && operation.Conditional {
				conditional = true
			}
		}
	}
	if !conditional {
		return check.exprList(call.Args)
	}
	args := make([]*operand, len(call.Args))
	var context Type
	for i, expression := range call.Args {
		if marker, ok := ast.Unparen(expression).(*ast.CallExpr); ok {
			if operation, found := check.tgoOperation(marker); found && operation.Conditional {
				continue
			}
		}
		args[i] = new(operand)
		check.expr(nil, args[i], expression)
		if args[i].isValid() && !isUntyped(args[i].typ()) && context == nil {
			context = args[i].typ()
		}
	}
	for i, expression := range call.Args {
		if args[i] == nil {
			check.expressionType(expression, context)
			args[i] = new(operand)
			check.expr(nil, args[i], expression)
		}
	}
	return args
}

func deleteKeyType(typ Type) Type {
	var key Type
	if !underIs(typ, func(underlying Type) bool {
		mapping, ok := underlying.(*Map)
		if !ok || (key != nil && !Identical(key, mapping.key)) {
			return false
		}
		key = mapping.key
		return true
	}) {
		return nil
	}
	return key
}

func (check *Checker) conditionalExpr(target *target, x *operand, call *ast.CallExpr, hint Type) exprKind {
	// Branch statements can change state even without a call or receive.
	// Array len, cap, and range must therefore evaluate a conditional value.
	check.hasCallOrRecv = true
	x.expr = call
	x.typ_ = Typ[Invalid]
	body := call.Args[0].(*ast.FuncLit).Body.List[0]
	context := hint
	if target != nil {
		context = target.typ
	}
	values := check.conditionalStatement(body, context)
	if check.conditionalValues == nil {
		check.conditionalValues = make(map[*ast.CallExpr][]*operand)
	}
	check.conditionalValues[call] = values
	for _, branch := range values {
		if !branch.isValid() {
			return expression
		}
	}
	// Generic calls determine the parameter type from all branch values.
	if context == nil && check.inferConditionals[call] {
		typ := Type(Typ[UntypedNil])
		if len(values) != 0 {
			typ = values[0].typ()
		}
		x.mode_, x.typ_ = value, typ
		return expression
	}
	// A byte-slice source of copy or append can instead be a string.
	// Keep its own string type when every value branch produces a string.
	if context != nil && check.sliceSourceContext != nil && Identical(context, check.sliceSourceContext) && len(values) != 0 {
		strings := true
		for _, branch := range values {
			strings = strings && allString(branch.typ())
		}
		if strings {
			context = nil
		}
	}
	if context != nil {
		for _, branch := range values {
			check.assignment(branch, context, "conditional expression branch")
			if !branch.isValid() {
				return expression
			}
		}
		x.mode_, x.typ_ = value, context
		return expression
	}
	var typ Type
	for _, branch := range values {
		candidate := branch.typ()
		if isUntyped(candidate) {
			continue
		}
		if typ != nil && !Identical(typ, candidate) {
			check.error(call, IncompatibleAssign, "conditional branch types differ; add an explicit type annotation")
			return expression
		}
		typ = candidate
	}
	if typ != nil {
		check.convertConditional(call, typ)
		for _, branch := range values {
			if !branch.isValid() {
				return expression
			}
		}
	} else {
		typ = untypedConditionalType(values)
	}
	if typ == nil {
		check.error(call, IncompatibleAssign, "conditional result type is unknown; add an explicit type annotation")
		return expression
	}
	x.mode_, x.typ_ = value, typ
	return expression
}

func untypedConditionalType(values []*operand) Type {
	var typ Type
	for _, branch := range values {
		if branch.isNil() {
			continue
		}
		if typ == nil {
			typ = branch.typ()
		} else {
			typ = maxType(typ, branch.typ())
			if typ == nil {
				break
			}
		}
	}
	for _, branch := range values {
		if branch.isNil() && typ != nil {
			typ = nil
			break
		}
	}
	return typ
}

func (check *Checker) convertConditional(call *ast.CallExpr, typ Type) {
	for _, branch := range check.conditionalValues[call] {
		check.assignment(branch, typ, "conditional expression branch")
	}
}

func (check *Checker) conditionalBranches(statement *ast.IfStmt, context Type) []*operand {
	previous := check.statementContext
	check.statementContext &^= fallthroughOk | finalSwitchCase | inTypeSwitch
	defer func() { check.statementContext = previous }()
	packageBranch := check.sig == nil && check.scope.Parent() == check.pkg.scope
	check.openScope(statement, "if expression")
	defer check.closeScope()
	if packageBranch {
		defer check.usage(check.scope)
	}
	if statement.Init != nil {
		check.stmt(check.statementContext, statement.Init)
	}
	var condition operand
	check.expr(nil, &condition, statement.Cond)
	if condition.isValid() && !allBoolean(condition.typ()) {
		check.error(statement.Cond, InvalidCond, "if expression condition must be Boolean")
	}
	values := check.conditionalBlock(statement.Body, context)
	switch alternative := statement.Else.(type) {
	case *ast.BlockStmt:
		values = append(values, check.conditionalBlock(alternative, context)...)
	case *ast.IfStmt:
		values = append(values, check.conditionalBranches(alternative, context)...)
	default:
		check.error(statement, InvalidSyntaxTree, "if expression requires else")
		values = append(values, &operand{expr: statement.Cond, typ_: Typ[Invalid]})
	}
	return values
}

func (check *Checker) conditionalBlock(body *ast.BlockStmt, context Type) []*operand {
	check.openScope(body, "conditional expression branch")
	defer check.closeScope()
	return check.conditionalBlockValues(body, context)
}

func (check *Checker) conditionalBlockValues(body *ast.BlockStmt, context Type) []*operand {
	list := trimTrailingEmptyStmts(body.List)
	if len(list) == 0 {
		check.error(body, InvalidSyntaxTree, "conditional expression branch requires a final value")
		return []*operand{{expr: &ast.BadExpr{From: body.Pos()}, typ_: Typ[Invalid]}}
	}
	check.stmtList(check.statementContext, list[:len(list)-1])
	flow := check.branchFlow(body)
	if !flow.Tail {
		context = nil
	}
	values := check.conditionalTail(list[len(list)-1], context, flow.Tail)
	flow = check.branchFlow(body)
	if check.Info != nil {
		if check.TGoBranches == nil {
			check.TGoBranches = make(map[*ast.BlockStmt]bool)
		}
		check.TGoBranches[body] = flow.End
	}
	if !flow.End {
		return nil
	}
	return values
}

func (check *Checker) branchFlow(body *ast.BlockStmt) controlflow.Result {
	return controlflow.Branch(body, controlflow.Syntax{
		RangeSourceOutside: func(loop *ast.RangeStmt) bool { return check.comprehensionRanges[loop] },
		Statements:         check.operationStatements,
		Terminates:         func(call *ast.CallExpr) bool { return check.isPanic[call] },
		Unevaluated:        func(call *ast.CallExpr) bool { return check.Types[call].Value != nil },
	})
}

func (check *Checker) conditionalTail(original ast.Stmt, context Type, reachable bool) []*operand {
	last := original
	for {
		labelled, ok := last.(*ast.LabeledStmt)
		if !ok {
			break
		}
		check.hasLabel = true
		last = labelled.Stmt
	}
	switch last.(type) {
	case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		return check.conditionalStatement(last, context)
	}
	if statement, ok := last.(*ast.ExprStmt); ok {
		branch := new(operand)
		check.rawExpr(newTarget(context, "conditional expression branch"), branch, statement.X, nil, false)
		check.exclude(branch, 1<<builtin|1<<typexpr)
		if branch.mode() == novalue && check.isTerminating(statement, "") {
			return nil
		}
		check.singleValue(branch)
		if branch.mode() == novalue && reachable {
			check.error(statement, InvalidSyntaxTree, "conditional expression branch requires one value")
			branch.invalidate()
		}
		return []*operand{branch}
	}
	check.stmt(check.statementContext, original)
	if check.isTerminating(original, "") || !reachable {
		return nil
	}
	if branch, ok := last.(*ast.BranchStmt); ok && (branch.Tok == token.BREAK || branch.Tok == token.CONTINUE) {
		return nil
	}
	check.error(last, InvalidSyntaxTree, "conditional expression branch requires a final value")
	return []*operand{{expr: &ast.BadExpr{From: last.Pos()}, typ_: Typ[Invalid]}}
}

func (check *Checker) propagationExpr(x *operand, call *ast.CallExpr, operation TGoOperation) exprKind {
	position := atPos(check.fset.File(call.Pos()).Pos(operation.Position.Offset))
	if conditional, ok := ast.Unparen(call.Args[0]).(*ast.CallExpr); ok {
		if op, found := check.tgoOperation(conditional); found && op.Conditional {
			check.error(position, InvalidSyntaxTree, "propagation requires a call")
			x.invalidate()
			return expression
		}
	}
	check.rawExpr(nil, x, call.Args[0], nil, false)
	x.expr = call
	if !x.isValid() {
		return expression
	}
	results, hasTupleResults := x.typ().(*Tuple)
	errorResultType := x.typ()
	if hasTupleResults && results.Len() != 0 {
		errorResultType = results.At(results.Len() - 1).Type()
	}
	if !AssignableTo(errorResultType, Universe.Lookup("error").Type()) {
		check.error(position, IncompatibleAssign, "propagation requires a final error result")
		x.invalidate()
		return expression
	}
	if check.sig == nil || check.sig.Results().Len() == 0 {
		check.error(position, IncompatibleAssign, "propagation requires a final error interface return")
		x.invalidate()
		return expression
	}
	returnedErrorType := errorResultType
	operator := "!"
	if operation.Wrap {
		returnedErrorType = Universe.Lookup("error").Type()
		operator = "!!"
	}
	returnErrorType := check.sig.Results().At(check.sig.Results().Len() - 1).Type()
	if _, ok := returnErrorType.Underlying().(*Interface); !ok || !AssignableTo(returnErrorType, Universe.Lookup("error").Type()) {
		check.error(position, IncompatibleAssign, "propagation requires a final error interface return")
		x.invalidate()
		return expression
	}
	if !AssignableTo(returnedErrorType, returnErrorType) {
		check.errorf(position, IncompatibleAssign, "%s cannot return error of type %s as %s", operator, returnedErrorType, returnErrorType)
		x.invalidate()
		return expression
	}
	x.mode_ = value
	switch {
	case !hasTupleResults || results.Len() == 1:
		x.mode_ = novalue
		x.typ_ = NewTuple()
	case results.Len() == 2:
		x.typ_ = results.At(0).Type()
	default:
		x.typ_ = NewTuple(results.vars[:results.Len()-1]...)
	}
	return statement
}

// expressionStatements visits source statements inside added expressions. Nested statements
// are visited by the caller; real function literals belong to another function.
func (check *Checker) expressionStatements(statement ast.Stmt, visit func(ast.Stmt)) {
	var first, second ast.Stmt
	switch statement := statement.(type) {
	case *ast.IfStmt:
		first = statement.Init
	case *ast.ForStmt:
		first, second = statement.Init, statement.Post
	case *ast.SwitchStmt:
		first = statement.Init
	case *ast.TypeSwitchStmt:
		first, second = statement.Init, statement.Assign
	case *ast.CommClause:
		first = statement.Comm
	}
	ast.Inspect(statement, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		if child, ok := node.(ast.Stmt); ok && child != statement && child != first && child != second {
			return false
		}
		if call, ok := node.(*ast.CallExpr); ok {
			if body := check.operationStatements(call); body != nil {
				visit(body)
				return false
			}
		}
		_, function := node.(*ast.FuncLit)
		return !function
	})
}

// prepareConditionalArgument records direct and nested branch values for inference.
func (check *Checker) prepareConditionalArgument(expr ast.Expr) {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok {
		return
	}
	operation, found := check.tgoOperation(call)
	if !found || !operation.Conditional {
		return
	}
	if check.inferConditionals == nil {
		check.inferConditionals = make(map[*ast.CallExpr]bool)
	}
	check.inferConditionals[call] = true
	check.prepareConditionalTails(call.Args[0].(*ast.FuncLit).Body.List[0])
}

func (check *Checker) prepareConditionalTails(statement ast.Stmt) {
	switch statement := statement.(type) {
	case *ast.IfStmt:
		check.prepareConditionalTails(statement.Body)
		if statement.Else != nil {
			check.prepareConditionalTails(statement.Else)
		}
	case *ast.SwitchStmt:
		for _, clause := range statement.Body.List {
			check.prepareConditionalTails(clause)
		}
	case *ast.TypeSwitchStmt:
		for _, clause := range statement.Body.List {
			check.prepareConditionalTails(clause)
		}
	case *ast.CaseClause:
		check.prepareConditionalTails(&ast.BlockStmt{List: statement.Body})
	case *ast.BlockStmt:
		if list := trimTrailingEmptyStmts(statement.List); len(list) != 0 {
			check.prepareConditionalTails(list[len(list)-1])
		}
	case *ast.LabeledStmt:
		check.prepareConditionalTails(statement.Stmt)
	case *ast.ExprStmt:
		check.prepareConditionalArgument(statement.X)
	}
}

// conditionalInference checks each completing branch with the same parameter.
// Terminating branches supply no operand; the other arguments must supply its type.
func (check *Checker) conditionalInference(params *Tuple, args []*operand) (*Tuple, []*operand) {
	var expandedParams []*Var
	var expandedArgs []*operand
	var add func(*Var, *operand)
	add = func(param *Var, arg *operand) {
		if call, ok := ast.Unparen(arg.expr).(*ast.CallExpr); ok && check.inferConditionals[call] && arg.isValid() {
			for _, branch := range check.conditionalValues[call] {
				add(param, branch)
			}
			return
		}
		expandedParams, expandedArgs = append(expandedParams, param), append(expandedArgs, arg)
	}
	for i, arg := range args {
		add(params.At(i), arg)
	}
	return NewTuple(expandedParams...), expandedArgs
}

// assignConditionalArgument checks each branch against the inferred parameter.
func (check *Checker) assignConditionalArgument(arg *operand, typ Type, context string) bool {
	call, ok := ast.Unparen(arg.expr).(*ast.CallExpr)
	if !ok || !check.inferConditionals[call] || !arg.isValid() {
		return false
	}
	for _, branch := range check.conditionalValues[call] {
		if !check.assignConditionalArgument(branch, typ, context) {
			check.assignment(branch, typ, context)
		}
		if !branch.isValid() {
			arg.invalidate()
			return true
		}
	}
	arg.mode_, arg.typ_, arg.val = value, typ, nil
	for expr := arg.expr; ; {
		delete(check.untyped, expr)
		check.recordTypeAndValue(expr, value, typ, nil)
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = paren.X
	}
	return true
}

// conditionalFunctionExit checks exits that Go's statement grammar cannot see.
func (check *Checker) conditionalFunctionExit(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if operation, exists := check.tgoOperation(call); exists && (operation.Conditional || operation.Comprehension) {
				found = true
				return false
			}
		}
		_, function := node.(*ast.FuncLit)
		return !function
	})
	if !found || check.branchFlow(body).End {
		return false
	}
	if check.Info != nil {
		if check.TGoReturnBlocks == nil {
			check.TGoReturnBlocks = make(map[*ast.BlockStmt]bool)
		}
		check.TGoReturnBlocks[body] = true
	}
	return true
}

// rejectRangeCollectionContinue checks jumps before a range has its collection.
func (check *Checker) rejectRangeCollectionContinue(expr ast.Expr) {
	ast.Walk(rangeContinueVisitor{check: check}, expr)
}

type rangeContinueVisitor struct {
	check *Checker
	loops int
}

func (visitor rangeContinueVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case nil:
		return nil
	case *ast.FuncLit:
		return nil
	case *ast.CallExpr:
		if body := visitor.check.operationStatements(node); body != nil {
			ast.Walk(visitor, body)
			return nil
		}
	case *ast.ForStmt, *ast.RangeStmt:
		visitor.loops++
	case *ast.BranchStmt:
		if node.Tok == token.CONTINUE && node.Label == nil && visitor.loops == 0 {
			visitor.check.error(node, MisplacedContinue, "continue cannot advance a range before its collection is computed")
		}
	}
	return visitor
}

type conditionalCase struct {
	context Type
	values  []*operand
}

func (check *Checker) conditionalStatement(statement ast.Stmt, context Type) []*operand {
	if statement, ok := statement.(*ast.IfStmt); ok {
		return check.conditionalBranches(statement, context)
	}
	var switchBody *ast.BlockStmt
	switch statement := statement.(type) {
	case *ast.SwitchStmt:
		switchBody = statement.Body
	case *ast.TypeSwitchStmt:
		switchBody = statement.Body
	}
	if check.conditionalCases == nil {
		check.conditionalCases = make(map[*ast.CaseClause]*conditionalCase)
	}
	hasDefaultCase := false
	for _, caseStatement := range switchBody.List {
		clause := caseStatement.(*ast.CaseClause)
		hasDefaultCase = hasDefaultCase || clause.List == nil
		check.conditionalCases[clause] = &conditionalCase{context: context}
	}
	if !hasDefaultCase {
		check.error(statement, InvalidSyntaxTree, "switch expression requires default")
	}
	// The first visited switch is the value switch, not a permitted break target.
	ast.Walk(valueSwitchBreakVisitor{check: check, nestedBreakTargets: -1}, statement)
	check.stmt(check.statementContext, statement)
	var branchValues []*operand
	for _, caseStatement := range switchBody.List {
		clause := caseStatement.(*ast.CaseClause)
		branchValues = append(branchValues, check.conditionalCases[clause].values...)
		delete(check.conditionalCases, clause)
	}
	if !hasDefaultCase {
		invalidBranch := &operand{expr: &ast.BadExpr{From: switchBody.Pos()}, typ_: Typ[Invalid]}
		branchValues = append(branchValues, invalidBranch)
	}
	return branchValues
}

func (check *Checker) conditionalCase(clause *ast.CaseClause, context Type) []*operand {
	enclosingContext := check.statementContext
	check.statementContext &^= breakOk | fallthroughOk | finalSwitchCase
	defer func() { check.statementContext = enclosingContext }()
	caseBody := &ast.BlockStmt{Lbrace: clause.Colon, List: clause.Body, Rbrace: clause.End()}
	branchValues := check.conditionalBlockValues(caseBody, context)
	if check.Info != nil {
		if check.TGoCases == nil {
			check.TGoCases = make(map[*ast.CaseClause]bool)
		}
		check.TGoCases[clause] = check.TGoBranches[caseBody]
	}
	return branchValues
}

type valueSwitchBreakVisitor struct {
	check              *Checker
	nestedBreakTargets int
}

func (visitor valueSwitchBreakVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case nil, *ast.FuncLit:
		return nil
	case *ast.CallExpr:
		if body := visitor.check.operationStatements(node); body != nil {
			ast.Walk(visitor, body)
			return nil
		}
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		visitor.nestedBreakTargets++
	case *ast.BranchStmt:
		if node.Tok == token.BREAK && node.Label == nil && visitor.nestedBreakTargets == 0 {
			visitor.check.error(node, InvalidSyntaxTree, "break cannot target a switch expression")
		}
	}
	return visitor
}
