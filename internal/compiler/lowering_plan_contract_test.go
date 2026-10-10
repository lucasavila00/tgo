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
	files := token.NewFileSet()
	inputSource, err := parseSource(files, "sample.tgo", []byte(`package sample

func check() (bool, error) { return true, nil }

func use(ready bool) (bool, error) {
	return ready && check()!!, nil
}
`))
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

	var function propagationFunction
	for _, candidate := range unit.loweringFunctions(inputSource) {
		if candidate.body.Pos() == inputSource.File.Decls[1].(*ast.FuncDecl).Body.Pos() {
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

func formatLoweringContractAST(t *testing.T, files *token.FileSet, file *ast.File) string {
	t.Helper()
	var output bytes.Buffer
	if err := format.Node(&output, files, file); err != nil {
		t.Fatal(err)
	}
	return output.String()
}
