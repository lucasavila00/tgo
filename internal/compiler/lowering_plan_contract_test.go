package compiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"testing"
)

func TestLoweringPlanKeepsGuardedTypedWorkSeparateFromSource(t *testing.T) {
	fixture := newLoweringPlanContractFixture(t)
	assertLoweringPlanPreservesSource(t, fixture)
	assertGuardedPropagationPlan(t, fixture.plan)
	assertGuardedPropagationEmission(t, fixture)
}

func TestExactComprehensionPlanOwnsAllocationAndIndexedStore(t *testing.T) {
	fixture := newLoweringPlanContractFixtureSource(t, `package sample
func use(values []int) []int {
	return []int{for index, value := range values { value + index }}
}
`)
	assertLoweringPlanPreservesSource(t, fixture)
	root := fixture.plan.root.operations[0].expressions[0]
	if root.exact == nil || root.exact.source.id == 0 || root.exact.length.id == 0 ||
		root.exact.index.id == 0 {
		t.Fatalf("exact comprehension lacks source, length, or index facts: %#v", root.exact)
	}
	assertExactAllocationPlan(t, root)
	assertExactRangeAndTerminalPlan(t, root)
	fixture.unit.info = newInfo()
	newLoweringEmitter(fixture.unit, fixture.source, fixture.plan).emit()
	fixture.unit.typecheck()
	if len(fixture.unit.typeErrors) != 0 {
		t.Fatalf("exact plan emission does not type-check: %v\n%s", fixture.unit.typeErrors,
			formatLoweringContractAST(t, fixture.files, fixture.source.File))
	}
}

func assertExactAllocationPlan(t *testing.T, root *plannedExpression) {
	t.Helper()
	length := root.work.operations[1]
	if length.kind != planLength || length.inputs[0] != root.exact.source.id ||
		length.outputs[0] != root.exact.length.id || length.source != nil {
		t.Fatalf("exact allocation does not use planned source length: %#v", length)
	}
	allocation := plannedExpressionForSource(root.work, root.exact.makeCall)
	if allocation == nil || len(allocation.operands) < 3 ||
		allocation.operands[2].kind != planValueExpression ||
		allocation.operands[2].value != root.exact.length.id {
		t.Fatalf("exact allocation does not consume the length ID: %#v", allocation)
	}
}

func assertExactRangeAndTerminalPlan(t *testing.T, root *plannedExpression) {
	t.Helper()
	rangeOperation, _ := plannedOperationForSource(root.work, root.exact.outer)
	terminal, _ := plannedOperationForSource(root.work, root.exact.assignment)
	if rangeOperation == nil || rangeOperation.rangeKey != root.exact.index.id ||
		rangeOperation.rangeKeyObject == nil || rangeOperation.source != root.exact.outer {
		t.Fatalf("exact range does not bind its source key object: %#v", rangeOperation)
	}
	if terminal == nil || terminal.assignment == nil || len(terminal.assignment.stores) != 1 ||
		!terminal.assignment.stores[0].place.preparedIndex ||
		terminal.expressions[0].source == root.exact.appendCall {
		t.Fatalf("exact terminal is not a planned indexed store: %#v", terminal)
	}
}

type loweringPlanContractFixture struct {
	files  *token.FileSet
	unit   *packageUnit
	source *source
	plan   *functionLoweringPlan
	before string
	defs   map[*ast.Ident]types.Object
	uses   map[*ast.Ident]types.Object
}

func newLoweringPlanContractFixture(t *testing.T) loweringPlanContractFixture {
	t.Helper()
	return newLoweringPlanContractFixtureSource(t, `package sample

func check() (bool, error) { return true, nil }

func use(ready bool) (bool, error) {
	return ready && check()!!, nil
}
`)
}

func newLoweringPlanContractFixtureSource(
	t *testing.T,
	text string,
) loweringPlanContractFixture {
	t.Helper()
	files := token.NewFileSet()
	inputSource, err := parseSource(files, "sample.tgo", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	unit := &packageUnit{
		Path: "sample", Sources: []*source{inputSource}, Files: []*ast.File{inputSource.File},
		Models: make(map[string]*model), Imports: make(map[string]*packageUnit),
		fs: files, importer: importer.Default(),
	}
	unit.prepare()
	unit.typecheck()

	var target *ast.FuncDecl
	for _, declaration := range inputSource.File.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "use" {
			target = function
			break
		}
	}
	if target == nil {
		t.Fatal("use function is not in the source")
	}
	var function propagationFunction
	for _, candidate := range unit.loweringFunctions(inputSource) {
		if candidate.body.Pos() == target.Body.Pos() {
			function = candidate
			break
		}
	}
	if function.body == nil {
		t.Fatal("use function is not in the lowering input")
	}
	before := formatLoweringContractAST(t, files, inputSource.File)
	defs := make(map[*ast.Ident]types.Object, len(unit.info.Defs))
	for identifier, object := range unit.info.Defs {
		defs[identifier] = object
	}
	uses := make(map[*ast.Ident]types.Object, len(unit.info.Uses))
	for identifier, object := range unit.info.Uses {
		uses[identifier] = object
	}
	plan := buildFunctionLoweringPlan(unit, inputSource, function)
	return loweringPlanContractFixture{
		files: files, unit: unit, source: inputSource, plan: plan,
		before: before, defs: defs, uses: uses,
	}
}

func assertLoweringPlanPreservesSource(t *testing.T, fixture loweringPlanContractFixture) {
	t.Helper()
	after := formatLoweringContractAST(t, fixture.files, fixture.source.File)
	if after != fixture.before {
		t.Fatalf("plan construction changed the source AST\nbefore:\n%s\nafter:\n%s",
			fixture.before, after)
	}
	for identifier, object := range fixture.defs {
		if fixture.unit.info.Defs[identifier] != object {
			t.Fatalf("plan construction changed the definition for %q", identifier.Name)
		}
	}
	for identifier, object := range fixture.uses {
		if fixture.unit.info.Uses[identifier] != object {
			t.Fatalf("plan construction changed the use for %q", identifier.Name)
		}
	}
}

func assertGuardedPropagationPlan(t *testing.T, plan *functionLoweringPlan) {
	t.Helper()
	root := plan.root.operations[0]
	binary := root.expressions[0]
	if binary.kind != planBinaryExpression || len(binary.work.operations) != 2 {
		t.Fatalf("root expression does not have a binary work region: %#v", binary)
	}
	guard := binary.work.operations[1]
	if guard.kind != planBranch || guard.operator != token.LAND ||
		guard.body == nil || len(guard.body.operations) != 1 {
		t.Fatalf("logical AND does not own one guard child: %#v", guard)
	}
	store := guard.body.operations[0]
	if store.kind != planStore || len(store.expressions) != 1 {
		t.Fatalf("guard child does not contain the RHS store: %#v", store)
	}
	propagation := store.expressions[0]
	assertPropagationWork(t, plan, propagation)
}

func assertPropagationWork(
	t *testing.T,
	plan *functionLoweringPlan,
	propagation *plannedExpression,
) {
	t.Helper()
	if propagation.kind != planPropagationExpression || propagation.work == nil ||
		len(propagation.work.operations) != 2 {
		t.Fatalf("guarded RHS does not contain propagation work: %#v", propagation)
	}
	bind := propagation.work.operations[0]
	errorBranch := propagation.work.operations[1]
	if bind.kind != planBind || len(bind.outputs) != 2 ||
		errorBranch.kind != planBranch || errorBranch.body == nil ||
		len(errorBranch.body.operations) != 1 ||
		errorBranch.body.operations[0].kind != planReturn {
		t.Fatalf("propagation does not bind and return inside the guard: "+
			"bind=%#v branch=%#v", bind, errorBranch)
	}
	errorType := types.Universe.Lookup("error").Type()
	if len(propagation.results) != 1 ||
		!types.Identical(propagation.results[0].typ, types.Typ[types.Bool]) ||
		!types.Identical(plan.values[bind.outputs[1]-1].typ, errorType) {
		t.Fatalf("call result types are not bool and error: results=%v values=%v",
			propagation.results, plan.values)
	}
}

func assertGuardedPropagationEmission(t *testing.T, fixture loweringPlanContractFixture) {
	t.Helper()
	checkedInfo := fixture.unit.info
	fixture.unit.info = newInfo()
	newLoweringEmitter(fixture.unit, fixture.source, fixture.plan).emit()
	fixture.unit.info = checkedInfo
	use := fixture.source.File.Decls[1].(*ast.FuncDecl)
	callCount, guardedCallCount := countGuardedCheckCalls(use)
	if callCount != 1 || guardedCallCount != 1 {
		t.Fatalf("emitter placed check calls outside the guard: calls=%d guarded=%d\n%s",
			callCount, guardedCallCount,
			formatLoweringContractAST(t, fixture.files, fixture.source.File))
	}
}

func countGuardedCheckCalls(use *ast.FuncDecl) (int, int) {
	callCount := 0
	guardedCallCount := 0
	for _, statement := range use.Body.List {
		insideGuard := false
		if conditional, ok := statement.(*ast.IfStmt); ok {
			insideGuard = true
			ast.Inspect(conditional.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if ok {
					if identifier, named := call.Fun.(*ast.Ident); named &&
						identifier.Name == "check" {
						callCount++
						guardedCallCount++
					}
				}
				return true
			})
		}
		if insideGuard {
			continue
		}
		ast.Inspect(statement, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok {
				if identifier, named := call.Fun.(*ast.Ident); named && identifier.Name == "check" {
					callCount++
				}
			}
			return true
		})
	}
	return callCount, guardedCallCount
}

func TestLoweringPlanSeparatesForeignBooleanNormalization(t *testing.T) {
	library := types.NewPackage("example.com/hidden", "hidden")
	hiddenName := types.NewTypeName(token.NoPos, library, "hiddenBool", nil)
	hidden := types.NewNamed(hiddenName, types.Typ[types.Bool], nil)
	library.Scope().Insert(hiddenName)
	library.Scope().Insert(types.NewFunc(
		token.NoPos,
		library,
		"Check",
		types.NewSignatureType(
			nil, nil, nil, types.NewTuple(),
			types.NewTuple(
				types.NewVar(token.NoPos, library, "", hidden),
				types.NewVar(token.NoPos, library, "", types.Universe.Lookup("error").Type()),
			),
			false,
		),
	))
	library.Scope().Insert(types.NewFunc(
		token.NoPos,
		library,
		"Consume",
		types.NewSignatureType(
			nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, library, "", hidden),
				types.NewVar(token.NoPos, library, "", types.Typ[types.Int]),
			),
			types.NewTuple(types.NewVar(token.NoPos, library, "", types.Typ[types.Int])),
			false,
		),
	))
	library.MarkComplete()

	plan := buildForeignContractPlan(t, library, `package sample

import "example.com/hidden"

func load() (int, error) { return 7, nil }

func use(ready bool) (int, error) {
	return hidden.Consume(ready && hidden.Check()!!, load()!!), nil
}
`)
	assertForeignBooleanNormalizationPlan(t, plan, hidden)
}

func assertForeignBooleanNormalizationPlan(
	t *testing.T,
	plan *functionLoweringPlan,
	hidden types.Type,
) {
	t.Helper()
	propagation := foreignBooleanPropagation(t, plan)
	bind := propagation.work.operations[0]
	errorBranch := propagation.work.operations[1]
	conversion := propagation.work.operations[2]
	sourceID := bind.outputs[0]
	convertedID := conversion.outputs[0]
	if bind.kind != planBind || !types.Identical(plan.values[sourceID-1].typ, hidden) {
		t.Fatalf("Check result lost its foreign type: bind=%#v value=%#v",
			bind, plan.values[sourceID-1])
	}
	if errorBranch.kind != planBranch || conversion.kind != planBooleanConvert ||
		conversion.inputs[0] != sourceID || convertedID == sourceID ||
		!types.Identical(plan.values[convertedID-1].typ, types.Typ[types.Bool]) {
		t.Fatalf("foreign result does not convert after its error branch: "+
			"branch=%#v conversion=%#v", errorBranch, conversion)
	}
	if len(propagation.results) != 1 || propagation.results[0].id != convertedID {
		t.Fatalf("logical RHS does not use the converted bool: results=%#v conversion=%#v",
			propagation.results, conversion)
	}
}

func foreignBooleanPropagation(t *testing.T, plan *functionLoweringPlan) *plannedExpression {
	t.Helper()
	call := plan.root.operations[0].expressions[0]
	if len(call.operands) < 3 {
		t.Fatalf("Consume call does not retain its operands: %#v", call)
	}
	logical := call.operands[1]
	if logical.work == nil || len(logical.work.operations) != 2 ||
		logical.work.operations[1].body == nil ||
		len(logical.work.operations[1].body.operations) != 1 {
		t.Fatalf("logical expression does not retain its guarded RHS: %#v", logical)
	}
	store := logical.work.operations[1].body.operations[0]
	propagation := store.expressions[0]
	if propagation.kind != planPropagationExpression || propagation.work == nil ||
		len(propagation.work.operations) != 3 {
		t.Fatalf("logical RHS does not have bind, error, and conversion work: %#v",
			propagation)
	}
	return propagation
}

func TestLoweringPlanKeepsOrdinaryForeignResultType(t *testing.T) {
	library := types.NewPackage("example.com/hidden", "hidden")
	hiddenName := types.NewTypeName(token.NoPos, library, "hiddenValue", nil)
	hidden := types.NewNamed(hiddenName, types.Typ[types.Int], nil)
	library.Scope().Insert(hiddenName)
	library.Scope().Insert(types.NewFunc(
		token.NoPos, library, "Factory",
		types.NewSignatureType(
			nil, nil, nil, types.NewTuple(),
			types.NewTuple(types.NewVar(token.NoPos, library, "", hidden)), false,
		),
	))
	anyType := types.Universe.Lookup("any").Type()
	library.Scope().Insert(types.NewFunc(
		token.NoPos, library, "ConsumeAny",
		types.NewSignatureType(
			nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, library, "", anyType),
				types.NewVar(token.NoPos, library, "", types.Typ[types.Int]),
			),
			types.NewTuple(types.NewVar(token.NoPos, library, "", types.Typ[types.Int])),
			false,
		),
	))
	library.MarkComplete()

	plan := buildForeignContractPlan(t, library, `package sample

import "example.com/hidden"

func load() (int, error) { return 7, nil }

func use() (int, error) {
	return hidden.ConsumeAny(hidden.Factory(), load()!!), nil
}
`)
	call := plan.root.operations[0].expressions[0]
	if len(call.operands) < 3 {
		t.Fatalf("ConsumeAny call does not retain its operands: %#v", call)
	}
	factory := call.operands[1]
	if factory.materialized == 0 {
		t.Fatalf("ordinary call is not materialized before later work: %#v", factory)
	}
	if !types.Identical(factory.expected, anyType) || !types.Identical(factory.typ, hidden) ||
		!types.Identical(plan.values[factory.materialized-1].typ, hidden) {
		t.Fatalf("ordinary call lost its natural foreign type: expression=%#v value=%#v",
			factory, plan.values[factory.materialized-1])
	}
}

func TestLoweringPlanKeepsLogicalResultTypeBeforeInterfaceUse(t *testing.T) {
	library := types.NewPackage("example.com/hidden", "hidden")
	hiddenName := types.NewTypeName(token.NoPos, library, "hiddenBool", nil)
	hidden := types.NewNamed(hiddenName, types.Typ[types.Bool], nil)
	library.Scope().Insert(hiddenName)
	library.Scope().Insert(types.NewFunc(
		token.NoPos, library, "Factory",
		types.NewSignatureType(
			nil, nil, nil, types.NewTuple(),
			types.NewTuple(types.NewVar(token.NoPos, library, "", hidden)), false,
		),
	))
	anyType := types.Universe.Lookup("any").Type()
	library.Scope().Insert(types.NewFunc(
		token.NoPos, library, "ConsumeAny",
		types.NewSignatureType(
			nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, library, "", anyType),
				types.NewVar(token.NoPos, library, "", types.Typ[types.Int]),
			),
			types.NewTuple(types.NewVar(token.NoPos, library, "", types.Typ[types.Int])),
			false,
		),
	))
	library.MarkComplete()

	plan := buildForeignContractPlan(t, library, `package sample

import "example.com/hidden"

func load() (int, error) { return 7, nil }

func use() (int, error) {
	return hidden.ConsumeAny(hidden.Factory() && hidden.Factory(), load()!!), nil
}
`)
	call := plan.root.operations[0].expressions[0]
	logical := call.operands[1]
	if !types.Identical(logical.expected, anyType) || len(logical.results) != 1 ||
		!types.Identical(logical.results[0].typ, hidden) {
		t.Fatalf("logical result lost its natural type: %#v", logical)
	}
	resultID := logical.results[0].id
	if logical.work == nil || len(logical.work.operations) != 2 ||
		logical.work.operations[0].outputs[0] != resultID ||
		logical.work.operations[1].inputs[0] != resultID ||
		!types.Identical(plan.values[resultID-1].typ, hidden) {
		t.Fatalf("logical branch does not use its typed result: %#v", logical.work)
	}
}

func TestLoweringReportsInvalidSliceComprehensionAtSource(t *testing.T) {
	_, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "input.tgo", Data: []byte(`package sample

func values(input []int) map[int]int {
	return map[int]int{for _, value := range input { value }}
}
`)}},
		FileSet:  token.NewFileSet(),
		Importer: importer.Default(),
	})
	if len(problems) != 1 ||
		problems[0].Error() != "input.tgo:4:9: slice comprehension needs a slice output type" {
		t.Fatalf("invalid slice comprehension problems: %v", problems)
	}
}

func TestLoweringReportsInvalidMapComprehensionAtSource(t *testing.T) {
	_, problems := Compile(PackageInput{
		Path: "sample",
		Sources: []File{{Name: "input.tgo", Data: []byte(`package sample

func values(input []int) []int {
	return []int{for _, value := range input { value: value }}
}
`)}},
		FileSet:  token.NewFileSet(),
		Importer: importer.Default(),
	})
	if len(problems) != 1 ||
		problems[0].Error() != "input.tgo:4:9: map comprehension needs a map output type" {
		t.Fatalf("invalid map comprehension problems: %v", problems)
	}
}

func buildForeignContractPlan(
	t *testing.T,
	library *types.Package,
	sourceText string,
) *functionLoweringPlan {
	t.Helper()
	files := token.NewFileSet()
	input, err := parseSource(files, "sample.tgo", []byte(sourceText))
	if err != nil {
		t.Fatal(err)
	}
	unit := &packageUnit{
		Path: "sample", Sources: []*source{input}, Files: []*ast.File{input.File},
		Models: make(map[string]*model), Imports: make(map[string]*packageUnit),
		fs: files,
		importer: packageImporter{
			"example.com/hidden": library,
		},
	}
	unit.prepare()
	unit.typecheck()
	var target *ast.FuncDecl
	for _, declaration := range input.File.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "use" {
			target = function
			break
		}
	}
	if target == nil {
		t.Fatal("use function is not in the source")
	}
	for _, function := range unit.loweringFunctions(input) {
		if function.body.Pos() == target.Body.Pos() {
			return buildFunctionLoweringPlan(unit, input, function)
		}
	}
	t.Fatal("use function is not in the lowering input")
	return nil
}

func formatLoweringContractAST(t *testing.T, files *token.FileSet, file *ast.File) string {
	t.Helper()
	var output bytes.Buffer
	if err := format.Node(&output, files, file); err != nil {
		t.Fatal(err)
	}
	return output.String()
}
