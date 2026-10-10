package compiler

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestAssignmentPlanEmissionUsesOnlyCapturedFacts(t *testing.T) {
	fixture := newLoweringPlanContractFixtureSource(t, `package sample

func load() (int, error) { return 7, nil }
type mixed interface { ~[2]int | ~[]int }

func use[S mixed](values S) error {
	values[0] = load()!!
	return nil
}
`)
	assertLoweringPlanPreservesSource(t, fixture)
	checkedInfo := fixture.unit.info
	fixture.unit.info = newInfo()
	newLoweringEmitter(fixture.unit, fixture.source, fixture.plan).emit()
	fixture.unit.info = checkedInfo

	output := formatLoweringContractAST(t, fixture.files, fixture.source.File)
	generatedFiles := token.NewFileSet()
	generated, err := parser.ParseFile(generatedFiles, "sample.go", output, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse emitted assignment: %v\n%s", err, output)
	}
	configuration := types.Config{Importer: importer.Default()}
	if _, err := configuration.Check(
		"sample", generatedFiles, []*ast.File{generated}, nil,
	); err != nil {
		t.Fatalf("type-check emitted assignment: %v\n%s", err, output)
	}
}

func TestAssignmentPlanOwnsPreparationAndRHSArity(t *testing.T) {
	plan := buildForeignContractPlan(t, types.NewPackage("unused", "unused"), `package sample

func pair() (int, string) { return 1, "one" }
func pairFallible() (int, string, error) { return 1, "one", nil }
func index() (int, error) { return 0, nil }
func key() (string, error) { return "key", nil }
func anyValue() (any, error) { return 1, nil }

type mixed interface { ~[2]int | ~[]int }

func use[S mixed](values S, mapped map[string]int, reused string) error {
	var number int
	var text string
	var ok bool
	values[index()!!], text = pair()
	number, ok = mapped[key()!!]
	number, ok = (anyValue()!!).(int)
	created, reused := pairFallible()!!
	_, _, _, _ = number, text, ok, created
	return nil
}
`)
	assignments := directAssignmentPlans(plan.root)
	if len(assignments) < 4 {
		t.Fatalf("direct assignment plans=%d want at least 4", len(assignments))
	}

	tuple := assignments[0]
	assertAlternativePreparation(t, tuple)
	assertAssignmentRightTypes(t, tuple, []types.Type{
		types.Typ[types.Int], types.Typ[types.String],
	})
	if len(tuple.stores) != 2 {
		t.Fatalf("tuple stores=%d want 2", len(tuple.stores))
	}

	assertAssignmentRightTypes(t, assignments[1], []types.Type{
		types.Typ[types.Int], types.Typ[types.Bool],
	})
	assertAssignmentRightTypes(t, assignments[2], []types.Type{
		types.Typ[types.Int], types.Typ[types.Bool],
	})
	assertDefinitionTargets(t, assignments[3])
}

func TestSelectedReceiveAssignmentOwnsChosenPreparation(t *testing.T) {
	plan := buildForeignContractPlan(t, types.NewPackage("unused", "unused"), `package sample

func index() (int, error) { return 0, nil }
type mixed interface { ~[2]int | ~[]int }

func use[S mixed](values S, input <-chan int) error {
	select {
	case values[index()!!] = <-input:
	}
	return nil
}
`)
	var selected *plannedAssignment
	walkPlanOperations(plan.root, func(operation *plannedOperation) {
		for _, communication := range operation.communications {
			if communication.assignment != nil {
				selected = communication.assignment
			}
		}
	})
	if selected == nil {
		t.Fatal("selected receive assignment is not planned")
	}
	assertAlternativePreparation(t, selected)
	if selected.rhs != nil || len(selected.right) != 0 || len(selected.stores) != 1 {
		t.Fatalf("selected assignment has unexpected RHS or stores: %#v", selected)
	}
	if selected.stores[0].value.typ == nil ||
		!types.Identical(selected.stores[0].value.typ, types.Typ[types.Int]) {
		t.Fatalf("selected receive value type=%v want int", selected.stores[0].value.typ)
	}
}

func assertAlternativePreparation(t *testing.T, assignment *plannedAssignment) {
	t.Helper()
	block := assignment.prepare
	assertAlternativePreparationKinds(t, block)
	branch := block.operations[2]
	declareID := block.operations[0].outputs[0]
	kindID := block.operations[1].outputs[0]
	if branch.operator != token.LOR || branch.body == nil ||
		len(branch.body.operations) != 1 ||
		branch.body.operations[0].kind != planLoadPlace {
		t.Fatalf("conditional snapshot is not owned by preparation: %#v", branch)
	}
	load := branch.body.operations[0]
	if len(branch.inputs) != 1 || branch.inputs[0] != kindID ||
		len(load.outputs) != 1 || load.outputs[0] != declareID {
		t.Fatalf("conditional snapshot IDs do not dominate: declare=%v kind=%v branch=%#v",
			declareID, kindID, branch)
	}
	alternative := assignmentAlternative(t, assignment.places)
	if alternative == nil {
		t.Fatalf("assignment does not retain an alternative place: %#v", assignment.places)
	}
	if alternative.snapshot.id != declareID || alternative.isArray.id != kindID ||
		block.operations[3].outputs[0] != alternative.index.id {
		t.Fatalf("place alternative IDs do not match preparation: %#v", alternative)
	}
}

func assertAlternativePreparationKinds(t *testing.T, block *plannedBlock) {
	t.Helper()
	if block == nil || len(block.operations) != 4 {
		t.Fatalf("preparation operations=%v want declare, kind, branch, index", block)
	}
	want := []plannedOperationKind{
		planDeclareValue, planTypeKind, planBranch, planEvaluate,
	}
	for index, kind := range want {
		if block.operations[index].kind != kind {
			t.Fatalf("preparation[%d]=%v want %v", index, block.operations[index].kind, kind)
		}
	}
}

func assignmentAlternative(t *testing.T, places []*plannedPlace) *plannedPlaceAlternative {
	t.Helper()
	var result *plannedPlaceAlternative
	for _, place := range places {
		if place.alternative == nil {
			continue
		}
		if result != nil {
			t.Fatalf("assignment has multiple alternative places: %#v", places)
		}
		result = place.alternative
	}
	return result
}

func assertAssignmentRightTypes(
	t *testing.T,
	assignment *plannedAssignment,
	want []types.Type,
) {
	t.Helper()
	if assignment == nil || len(assignment.right) != 1 || assignment.rhs == nil ||
		len(assignment.rhs.operations) != 1 {
		t.Fatalf("assignment RHS plan=%#v", assignment)
	}
	right := assignment.right[0]
	evaluate := assignment.rhs.operations[0]
	if right.required != len(want) || evaluate.resultCount != len(want) ||
		len(right.values) != len(want) || len(evaluate.outputs) != len(want) ||
		len(right.expected) != len(want) || len(evaluate.contexts) != len(want) {
		t.Fatalf("RHS arity right=%#v evaluate=%#v", right, evaluate)
	}
	for index, typ := range want {
		assertAssignmentRightType(t, right, evaluate, index, typ)
	}
}

func assertAssignmentRightType(
	t *testing.T,
	right plannedAssignmentRight,
	evaluate *plannedOperation,
	index int,
	want types.Type,
) {
	t.Helper()
	if right.values[index].id != evaluate.outputs[index] {
		t.Fatalf("RHS[%d] value ID=%v evaluate output=%v", index,
			right.values[index].id, evaluate.outputs[index])
	}
	if !types.Identical(right.values[index].typ, want) ||
		!types.Identical(right.expected[index].typ, want) ||
		!types.Identical(evaluate.contexts[index].typ, want) {
		t.Fatalf("RHS[%d] actual=%v expected=%v context=%v want=%v", index,
			right.values[index].typ, right.expected[index].typ,
			evaluate.contexts[index].typ, want)
	}
}

func assertDefinitionTargets(t *testing.T, assignment *plannedAssignment) {
	t.Helper()
	if assignment == nil || !assignment.define || len(assignment.targets) != 2 {
		t.Fatalf("definition assignment=%#v", assignment)
	}
	if !assignment.targets[0].define || assignment.targets[0].object == nil {
		t.Fatalf("new target is not a definition: %#v", assignment.targets[0])
	}
	if assignment.targets[1].define || assignment.targets[1].object == nil {
		t.Fatalf("reused target is not a use: %#v", assignment.targets[1])
	}
	first, firstOK := assignment.targets[0].source.(*ast.Ident)
	second, secondOK := assignment.targets[1].source.(*ast.Ident)
	if !firstOK || !secondOK || first.Name != "created" || second.Name != "reused" {
		t.Fatalf("definition target sources=%v, %v", first, second)
	}
}

func directAssignmentPlans(block *plannedBlock) []*plannedAssignment {
	var result []*plannedAssignment
	walkPlanOperations(block, func(operation *plannedOperation) {
		if operation.assignment != nil {
			result = append(result, operation.assignment)
		}
	})
	return result
}

func walkPlanOperations(block *plannedBlock, visit func(*plannedOperation)) {
	if block == nil {
		return
	}
	for _, operation := range block.operations {
		visit(operation)
		for _, child := range []*plannedBlock{
			operation.init, operation.test, operation.body, operation.post,
			operation.otherwise, operation.before, operation.after,
		} {
			walkPlanOperations(child, visit)
		}
		for _, child := range operation.cases {
			walkPlanOperations(child, visit)
		}
	}
}
