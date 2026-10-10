package app

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestPropagationSuccess(t *testing.T) {
	events := []string{}
	value, err := PropagationOuter(&events, true)
	if err != nil || value != 7 {
		t.Fatalf("value=%d error=%v", value, err)
	}
	if strings.Join(events, ",") != "load" {
		t.Fatalf("events=%v", events)
	}

	name, number, err := PropagationMany(true)
	if err != nil || name != "account" || number != 7 {
		t.Fatalf("pair=%q,%d error=%v", name, number, err)
	}
	name, number, err = PropagationManyDeclaration(true)
	if err != nil || name != "account" || number != 7 {
		t.Fatalf("declared pair=%q,%d error=%v", name, number, err)
	}
	if err := PropagationErrorOnly(true); err != nil {
		t.Fatalf("flush: %v", err)
	}
	generic, err := PropagationGeneric("safe", true)
	if err != nil || generic != "safe" {
		t.Fatalf("generic=%q error=%v", generic, err)
	}
}

func TestPropagationErrorChain(t *testing.T) {
	events := []string{}
	value, err := PropagationOuter(&events, false)
	if value != 0 {
		t.Fatalf("value=%d", value)
	}
	if !errors.Is(err, errPropagationMissing) {
		t.Fatalf("cause=%v", err)
	}
	want := "PropagationValue: propagationLoad: missing account"
	if err.Error() != want {
		t.Fatalf("error=%q want=%q", err, want)
	}
}

func TestTransparentPropagationReturnsOriginalError(t *testing.T) {
	events := []string{}
	value, err := PropagationTransparent(&events, false)
	if value != 0 || err != errPropagationMissing {
		t.Fatalf("value=%d error=%v", value, err)
	}
	value, err = PropagationTransparentFunction(&events, false)
	if value != 0 || err != errPropagationMissing {
		t.Fatalf("function value=%d error=%v", value, err)
	}

	events = nil
	text, err := PropagationTransparentNested(&events, true)
	if text != "beforeafter" || err != nil || strings.Join(events, ",") != "before,load,after" {
		t.Fatalf("text=%q error=%v events=%v", text, err, events)
	}
	events = nil
	text, err = PropagationTransparentNested(&events, false)
	if text != "" || err != errPropagationMissing || strings.Join(events, ",") != "before,load" {
		t.Fatalf("failed text=%q error=%v events=%v", text, err, events)
	}
}

func TestPropagationKeepsEvaluationOrder(t *testing.T) {
	events := []string{}
	value, err := PropagationNested(&events, true)
	if err != nil || value != "beforeafter" {
		t.Fatalf("value=%q error=%v", value, err)
	}
	if strings.Join(events, ",") != "before,load,after" {
		t.Fatalf("success events=%v", events)
	}

	events = nil
	value, err = PropagationNested(&events, false)
	if value != "" || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("value=%q error=%v", value, err)
	}
	if strings.Join(events, ",") != "before,load" {
		t.Fatalf("failure events=%v", events)
	}

	events = nil
	values, err := PropagationAssignment(&events, true)
	if err != nil || values[0] != 7 {
		t.Fatalf("assignment values=%v error=%v", values, err)
	}
	if strings.Join(events, ",") != "target,index,load" {
		t.Fatalf("assignment events=%v", events)
	}

	events = nil
	values, err = PropagationAssignment(&events, false)
	if values != nil || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("failed assignment values=%v error=%v", values, err)
	}
	if strings.Join(events, ",") != "target,index,load" {
		t.Fatalf("failed assignment events=%v", events)
	}

	events = nil
	number, err := PropagationIncrement(&events, true)
	if err != nil || number != 1 || strings.Join(events, ",") != "target,index" {
		t.Fatalf("increment value=%d error=%v events=%v", number, err, events)
	}

	events = nil
	number, err = PropagationIncrement(&events, false)
	if number != 0 || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("failed increment value=%d error=%v", number, err)
	}
	if strings.Join(events, ",") != "target" {
		t.Fatalf("failed increment events=%v", events)
	}

	number, err = PropagationArrayIndex(true)
	if err != nil || number != 2 {
		t.Fatalf("array index value=%d error=%v", number, err)
	}
	aliases, err := PropagationArraySlice(true)
	if err != nil || !aliases {
		t.Fatalf("array slice aliases=%v error=%v", aliases, err)
	}
	genericArray := [1]int{1}
	aliases, err = PropagationGenericArraySlice(&genericArray, true)
	if err != nil || !aliases {
		t.Fatalf("generic array slice aliases=%v error=%v", aliases, err)
	}
}

func TestPropagationUsesZeroResults(t *testing.T) {
	events := []string{}
	value, err := PropagationNamed(&events, false)
	if value != 0 || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("value=%d error=%v", value, err)
	}

	value, err = PropagationFunctionValue(&events, false)
	if value != 0 || !strings.HasPrefix(err.Error(), "load: ") {
		t.Fatalf("function value=%d error=%v", value, err)
	}

	text, err := PropagationGeneric("unsafe", false)
	if text != "" || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("generic=%q error=%v", text, err)
	}
}

func TestPropagationKeepsConditionalEvaluation(t *testing.T) {
	events := []string{}
	value, err := PropagationAnd(&events, false, true)
	if err != nil || value || len(events) != 0 {
		t.Fatalf("short and value=%v error=%v events=%v", value, err, events)
	}
	value, err = PropagationAnd(&events, true, true)
	if err != nil || !value || strings.Join(events, ",") != "right" {
		t.Fatalf("full and value=%v error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = PropagationOr(&events, true, false)
	if err != nil || !value || len(events) != 0 {
		t.Fatalf("short or value=%v error=%v events=%v", value, err, events)
	}
	value, err = PropagationOr(&events, false, true)
	if err != nil || !value || strings.Join(events, ",") != "right" {
		t.Fatalf("full or value=%v error=%v events=%v", value, err, events)
	}
}

func TestPropagationRunsInLoopCondition(t *testing.T) {
	count, err := PropagationLoop(0)
	if err != nil || count != 2 {
		t.Fatalf("count=%d error=%v", count, err)
	}
	count, err = PropagationLoop(2)
	if count != 0 || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("failed count=%d error=%v", count, err)
	}
}

func TestPropagationRunsForInitializer(t *testing.T) {
	events := []string{}
	count, err := PropagationForInitializer(&events, true, true)
	if err != nil || count != 2 {
		t.Fatalf("count=%d error=%v", count, err)
	}
	want := "first,second,condition,body,post,condition,body,post,condition,body"
	if strings.Join(events, ",") != want {
		t.Fatalf("events=%v want=%q", events, want)
	}

	events = nil
	count, err = PropagationForInitializer(&events, false, true)
	if count != 0 || err != errPropagationMissing || strings.Join(events, ",") != "first" {
		t.Fatalf("first failure count=%d error=%v events=%v", count, err, events)
	}

	events = nil
	count, err = PropagationForInitializer(&events, true, false)
	if count != 0 || err != errPropagationMissing || strings.Join(events, ",") != "first,second" {
		t.Fatalf("second failure count=%d error=%v events=%v", count, err, events)
	}

	events = nil
	count, err = PropagationForInitializerWrapped(&events, true)
	if count != 1 || err != nil || strings.Join(events, ",") != "wrapped" {
		t.Fatalf("wrapped success count=%d error=%v events=%v", count, err, events)
	}

	events = nil
	count, err = PropagationForInitializerWrapped(&events, false)
	if count != 0 || !errors.Is(err, errPropagationMissing) || err == errPropagationMissing {
		t.Fatalf("wrapped failure count=%d error=%v", count, err)
	}
	if err.Error() != "propagationForValue: missing account" {
		t.Fatalf("wrapped error=%q", err)
	}
}

func TestPropagationKeepsForInitializerLabels(t *testing.T) {
	for _, enterWithGoto := range []bool{false, true} {
		events := []string{}
		count, err := PropagationForInitializerLabels(&events, enterWithGoto)
		if err != nil || count != 1 || strings.Join(events, ",") != "label" {
			t.Fatalf("goto=%v count=%d error=%v events=%v", enterWithGoto, count, err, events)
		}
	}
}

func TestPropagationKeepsForInitializerIterationIdentity(t *testing.T) {
	closures, addresses, err := PropagationForInitializerIdentity()
	if err != nil {
		t.Fatal(err)
	}
	for index := range 3 {
		if closures[index]() != index || *addresses[index] != index {
			t.Fatalf("index=%d closure=%d address=%d", index, closures[index](), *addresses[index])
		}
		for other := range index {
			if addresses[index] == addresses[other] {
				t.Fatalf("iterations %d and %d have the same address", index, other)
			}
		}
	}
}

func TestPropagationInDeferredArgument(t *testing.T) {
	events := []string{}
	value, err := PropagationDeferredArgument(&events, true)
	if err != nil || value != 7 || strings.Join(events, ",") != "load,body,defer" {
		t.Fatalf("value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = PropagationDeferredArgument(&events, false)
	if value != 0 || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("value=%d error=%v", value, err)
	}
	if strings.Join(events, ",") != "load" {
		t.Fatalf("failure events=%v", events)
	}
}

func TestPropagationInControlExpressions(t *testing.T) {
	events := []string{}
	value, err := PropagationIf(&events, true)
	if err != nil || !value || strings.Join(events, ",") != "if" {
		t.Fatalf("if value=%v error=%v events=%v", value, err, events)
	}
	total, err := PropagationRange(true)
	if err != nil || total != 6 {
		t.Fatalf("range total=%d error=%v", total, err)
	}
	total, err = PropagationLabeledRange(true)
	if err != nil || total != 6 {
		t.Fatalf("labeled range total=%d error=%v", total, err)
	}
	label, err := PropagationSwitch(true)
	if err != nil || label != "two" {
		t.Fatalf("switch label=%q error=%v", label, err)
	}
	label, err = PropagationLabeledSwitch(true)
	if err != nil || label != "two" {
		t.Fatalf("labeled switch label=%q error=%v", label, err)
	}
	label, err = PropagationGotoLabeledSwitch(true)
	if err != nil || label != "two" {
		t.Fatalf("goto switch label=%q error=%v", label, err)
	}
	events = nil
	prefix, err := PropagationIfInitializer(&events, true)
	if err != nil || prefix != "init" || strings.Join(events, ",") != "init,if" {
		t.Fatalf("if init prefix=%q error=%v events=%v", prefix, err, events)
	}
	events = nil
	prefix, err = PropagationSwitchInitializer(&events, true)
	if err != nil || prefix != "init" || strings.Join(events, ",") != "init" {
		t.Fatalf("switch init prefix=%q error=%v events=%v", prefix, err, events)
	}
	events = nil
	label, err = PropagationTypeSwitch(&events, true)
	if err != nil || label != "init:ready" || strings.Join(events, ",") != "init,guard" {
		t.Fatalf("type switch label=%q error=%v events=%v", label, err, events)
	}
	events = nil
	label, err = PropagationTypeSwitchInitializer(&events, true)
	if err != nil || label != "7:ready" || strings.Join(events, ",") != "load" {
		t.Fatalf("type switch initializer label=%q error=%v events=%v", label, err, events)
	}
	events = nil
	label, err = PropagationTypeSwitchBoth(&events, true, true)
	if err != nil || label != "ready:ready" || strings.Join(events, ",") != "init,guard" {
		t.Fatalf("both type switch label=%q error=%v events=%v", label, err, events)
	}
	events = nil
	label, err = PropagationBareTypeSwitch(&events, true)
	if err != nil || label != "string" || strings.Join(events, ",") != "guard" {
		t.Fatalf("bare type switch label=%q error=%v events=%v", label, err, events)
	}
	events = nil
	label, err = PropagationLabeledTypeSwitch(&events, true)
	if err != nil || label != "ready" || strings.Join(events, ",") != "guard" {
		t.Fatalf("labeled type switch label=%q error=%v events=%v", label, err, events)
	}
	label, err = PropagationLabeledTypeSwitchScope()
	if err != nil || label != "outer" {
		t.Fatalf("labeled type switch scope label=%q error=%v", label, err)
	}
	events = nil
	label, err = PropagationGotoLabeledTypeSwitch(&events, true)
	if err != nil || label != "ready" || strings.Join(events, ",") != "guard" {
		t.Fatalf("goto type switch label=%q error=%v events=%v", label, err, events)
	}

	total, err = PropagationRange(false)
	if total != 0 || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("failed range total=%d error=%v", total, err)
	}
	label, err = PropagationSwitch(false)
	if label != "" || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("failed switch label=%q error=%v", label, err)
	}
	events = nil
	label, err = PropagationTypeSwitch(&events, false)
	if label != "" || err != errPropagationMissing ||
		!errors.Is(err, errPropagationMissing) || strings.Join(events, ",") != "init,guard" {
		t.Fatalf("failed type switch label=%q error=%v events=%v", label, err, events)
	}
	events = nil
	label, err = PropagationTypeSwitchInitializer(&events, false)
	if label != "" || err != errPropagationMissing ||
		!errors.Is(err, errPropagationMissing) || strings.Join(events, ",") != "load" {
		t.Fatalf("failed type switch initializer label=%q error=%v events=%v", label, err, events)
	}
	events = nil
	label, err = PropagationTypeSwitchBoth(&events, false, true)
	if label != "" || err != errPropagationMissing ||
		!errors.Is(err, errPropagationMissing) || strings.Join(events, ",") != "init" {
		t.Fatalf("failed type switch init label=%q error=%v events=%v", label, err, events)
	}
	events = nil
	label, err = PropagationTypeSwitchBoth(&events, true, false)
	if label != "" || err != errPropagationMissing ||
		!errors.Is(err, errPropagationMissing) || strings.Join(events, ",") != "init,guard" {
		t.Fatalf("failed type switch guard label=%q error=%v events=%v", label, err, events)
	}
}

func TestPropagationRunsExistingDefers(t *testing.T) {
	events := []string{}
	value, err := PropagationExistingDefer(&events, false)
	if value != 0 || !errors.Is(err, errPropagationMissing) {
		t.Fatalf("value=%d error=%v", value, err)
	}
	if strings.Join(events, ",") != "load,exit" {
		t.Fatalf("events=%v", events)
	}
}

func TestPropagationUsesGoTypedNilRule(t *testing.T) {
	value, err := PropagationTypedNil()
	if value != 0 || err == nil {
		t.Fatalf("value=%d error=%v", value, err)
	}
	var typed *propagationTypedError
	if !errors.As(err, &typed) {
		t.Fatalf("typed cause=%v", err)
	}
}

func TestPropagationGeneratedGoUsesDirectNames(t *testing.T) {
	data, err := os.ReadFile("propagation_tgo.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{
		"value, err := propagationLoad(events, found)",
		`return 0, fmt.Errorf("propagationLoad: %w", err)`,
		"func PropagationTransparent(events *[]string, found bool) (int, error) {\n\tvalue, err := propagationLoad(events, found)\n\tif err != nil {\n\t\treturn 0, err\n\t}",
		"var name, value, err = propagationPair(found)",
		"var zero T",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("generated Go lacks %q", required)
		}
	}
	if strings.Contains(text, "__tgo_") {
		t.Fatal("generated Go contains an old synthetic name")
	}
}
