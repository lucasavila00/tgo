package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

type valueID int
type placeID int
type scopeID int
type targetID int

type plannedValue struct {
	id            valueID
	typ           types.Type
	position      token.Pos
	explicit      bool
	typeReference plannedTypeReference
}

type plannedPlace struct {
	id            placeID
	typ           types.Type
	position      token.Pos
	kind          plannedPlaceKind
	source        ast.Expr
	object        types.Object
	base          *plannedPlace
	container     *plannedExpression
	index         *plannedExpression
	values        []plannedValue
	retainIndex   bool
	preparedIndex bool
	alternative   *plannedPlaceAlternative
}

type plannedPlaceKind uint8

const (
	planObjectPlace plannedPlaceKind = iota
	planDerefPlace
	planFieldPlace
	planArrayIndexPlace
	planSliceIndexPlace
	planMapIndexPlace
	planAlternativeIndexPlace
)

type plannedTypeKind uint8

const (
	planArrayType plannedTypeKind = iota + 1
)

type plannedPackageReference struct {
	path        string
	preferred   string
	importSpec  *ast.ImportSpec
	identifiers []*ast.Ident
}

type plannedBlock struct {
	scope       scopeID
	sourceScope *types.Scope
	operations  []*plannedOperation
}

type plannedTarget struct {
	id       targetID
	label    string
	position token.Pos
}

type plannedOperationKind uint8

const (
	planSourceStatement plannedOperationKind = iota
	planEvaluate
	planBind
	planStore
	planBranch
	planReturn
	planCopy
	planPlaceReady
	planBooleanConvert
	planDeclareValue
	planLoadPlace
	planTypeKind
	planJump
	planBlockStatement
	planIfStatement
	planRangeStatement
	planForStatement
	planSwitchStatement
	planTypeSwitchStatement
	planSelectStatement
	planLabeledStatement
)

type plannedOperation struct {
	kind           plannedOperationKind
	source         ast.Stmt
	target         targetID
	controlTarget  targetID
	sourceScope    *types.Scope
	expressions    []*plannedExpression
	places         []*plannedPlace
	init           *plannedBlock
	test           *plannedBlock
	body           *plannedBlock
	post           *plannedBlock
	otherwise      *plannedBlock
	cases          []*plannedBlock
	entry          []*plannedBlock
	selected       []*plannedBlock
	before         *plannedBlock
	inputs         []valueID
	outputs        []valueID
	resultCount    int
	contexts       []plannedTypeReference
	errorValue     valueID
	metadata       *propagationSource
	operator       token.Token
	preferred      string
	communications []*plannedCommunication
	headerBindings []plannedHeaderBinding
	binding        *plannedOperation
	after          *plannedBlock
	declarations   []*plannedDeclaration
	failureCommas  []token.Pos
	labelHasGoto   bool
	copyTarget     ast.Expr
	assignment     *plannedAssignment
	typeReference  plannedTypeReference
	typeKind       plannedTypeKind
	packageRef     plannedPackageReference
	declaresOutput bool
	rangeKey       valueID
}

type plannedDeclaration struct {
	source      *ast.ValueSpec
	expressions []*plannedExpression
	before      *plannedBlock
	binding     *plannedOperation
	after       *plannedBlock
}

type plannedCommunication struct {
	source        ast.Stmt
	channel       *plannedExpression
	value         *plannedExpression
	channelValue  plannedValue
	sendValue     plannedValue
	receiveValues []plannedValue
	targets       []*plannedPlace
	left          []ast.Expr
	token         token.Token
	assignment    *plannedAssignment
}

type plannedHeaderBinding struct {
	target ast.Expr
	value  plannedValue
}

type plannedExpressionKind uint8

const (
	planRetainedExpression plannedExpressionKind = iota
	planCallExpression
	planBinaryExpression
	planIndexExpression
	planIndexListExpression
	planSelectorExpression
	planSliceExpression
	planParenExpression
	planStarExpression
	planUnaryExpression
	planTypeAssertExpression
	planCompositeExpression
	planKeyValueExpression
	planPropagationExpression
	planComprehensionExpression
)

type plannedExpression struct {
	kind           plannedExpressionKind
	source         ast.Expr
	typ            types.Type
	expected       types.Type
	contextual     bool
	retainContext  bool
	resultCount    int
	results        []plannedValue
	operands       []*plannedExpression
	propagation    *propagationSource
	comprehension  *comprehensionSource
	function       *ast.FuncLit
	work           *plannedBlock
	before         *plannedBlock
	materialized   valueID
	booleanAdapter bool
	place          *plannedPlace
	resultNames    []*ast.Ident
	builtins       []*plannedBuiltin
	exact          *plannedExactComprehension
}

type plannedExactComprehension struct {
	makeCall   *ast.CallExpr
	outer      *ast.RangeStmt
	assignment *ast.AssignStmt
	appendCall *ast.CallExpr
	source     plannedValue
	position   token.Pos
	identity   bool
	index      plannedValue
}

type plannedBuiltin struct {
	call     *ast.CallExpr
	name     string
	position token.Pos
}

type functionLoweringPlan struct {
	function propagationFunction
	root     *plannedBlock
	values   []plannedValue
	places   []*plannedPlace
	targets  map[targetID]plannedTarget
}

type loweringPlanBuilder struct {
	unit         *packageUnit
	source       *source
	function     propagationFunction
	plan         *functionLoweringPlan
	nextScope    scopeID
	nextTarget   targetID
	bindings     map[types.Object]types.Type
	currentScope scopeID
	gotos        map[string]bool
	labels       map[string]targetID
	breakTargets []targetID
	loopTargets  []targetID
}

func buildFunctionLoweringPlan(
	unit *packageUnit,
	source *source,
	function propagationFunction,
) *functionLoweringPlan {
	plan := &functionLoweringPlan{
		function: function, targets: make(map[targetID]plannedTarget),
	}
	builder := &loweringPlanBuilder{
		unit: unit, source: source, function: function, plan: plan,
		bindings: make(map[types.Object]types.Type),
		gotos:    functionGotoLabels(function.body),
		labels:   make(map[string]targetID),
	}
	builder.indexLabels(function.body)
	plan.root = builder.block(function.body.List)
	return plan
}

func (b *loweringPlanBuilder) block(statements []ast.Stmt) *plannedBlock {
	outerBindings := b.bindings
	b.bindings = cloneTypeBindings(outerBindings)
	outerScope := b.currentScope
	b.nextScope++
	b.currentScope = b.nextScope
	block := &plannedBlock{scope: b.currentScope}
	if len(statements) != 0 {
		block.sourceScope = b.sourceScope(statements[0].Pos())
	}
	for _, statement := range statements {
		beforeStatement := cloneTypeBindings(b.bindings)
		block.operations = append(block.operations, b.statement(statement))
		if statementOwnsScope(statement) {
			b.bindings = beforeStatement
		} else {
			b.recordBindings(statement)
		}
	}
	b.bindings = outerBindings
	b.currentScope = outerScope
	return block
}

func statementOwnsScope(statement ast.Stmt) bool {
	switch statement.(type) {
	case *ast.BlockStmt, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt,
		*ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return true
	}
	return false
}

func cloneTypeBindings(input map[types.Object]types.Type) map[types.Object]types.Type {
	result := make(map[types.Object]types.Type, len(input))
	for object, typ := range input {
		result[object] = typ
	}
	return result
}

// statement defines the execution regions for one source statement.
