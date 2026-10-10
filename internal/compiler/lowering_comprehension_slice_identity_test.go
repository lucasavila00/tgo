package compiler

import (
	"strings"
	"testing"
)

func TestLoweringPreservesComprehensionAndSliceIdentity(t *testing.T) {
	t.Parallel()
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "comprehensionsliceidentity", sourceName: "identity.tgo",
		source: loweringComprehensionSliceIdentitySource, testName: "identity_test.tgo",
		testSource:  loweringComprehensionSliceIdentityTestSource,
		testPattern: "TestGeneratedComprehensionAndSliceIdentity",
		validate:    validateComprehensionSliceIdentityOutput,
	})
}

func validateComprehensionSliceIdentityOutput(t *testing.T, compiled *CompiledPackage) {
	t.Helper()
	output := string(compiled.Outputs["identity.tgo"])
	identityStart := strings.Index(output, "func identityCopy")
	transformedStart := strings.Index(output, "func transformedCopy")
	arrayStart := strings.Index(output, "func arrayRoot")
	if identityStart < 0 || transformedStart <= identityStart || arrayStart <= transformedStart {
		t.Fatalf("generated output lacks fixture functions\n%s", output)
	}
	identityOutput := output[identityStart:transformedStart]
	transformedOutput := output[transformedStart:arrayStart]
	if !strings.Contains(identityOutput, "copy(") {
		t.Fatalf("identity comprehension does not use copy\n%s", identityOutput)
	}
	if strings.Contains(transformedOutput, "copy(") ||
		!strings.Contains(transformedOutput, "] = transform(") {
		t.Fatalf("transformed comprehension lost indexed evaluation\n%s", transformedOutput)
	}
}

const loweringComprehensionSliceIdentitySource = `package comprehensionsliceidentity

import "errors"

var errBound = errors.New("bound failure")

func filtered(values []int) []int {
	return []int{for _, value := range values {
		if value % 2 != 0 {
			value
		}
	}}
}

func nested(values [][]int) []int {
	return []int{for _, group := range values {
		for _, value := range group {
			value
		}
	}}
}

func identityCopy(values []int) []int {
	return []int{for _, value := range values {
		value
	}}
}

func transform(value int) int { return value + 10 }

func transformedCopy(values []int) []int {
	return []int{for _, value := range values {
		transform(value)
	}}
}

func record(events *[]string, event string) {
	*events = append(*events, event)
}

func arrayRoot[T ~[1]int](events *[]string, values %T) %T {
	record(events, "root")
	return values
}

func arrayBound[T ~[1]int](events *[]string, values %T, fail bool) (int, error) {
	record(events, "bound")
	(*values)[0] = 2
	if fail {
		return 0, errBound
	}
	return 0, nil
}

func genericArraySlice[T ~[1]int](events *[]string, values %T, fail bool) ([]int, error) {
	result := (*arrayRoot(events, values))[arrayBound(events, values, fail)!!:]
	record(events, "after")
	return result, nil
}

func consumeArray[T ~[1]int](events *[]string, values T, index int) int {
	record(events, "consume")
	return values[index]
}

func genericArrayValue[T ~[1]int](events *[]string, values %T, fail bool) (int, error) {
	return consumeArray(events, *arrayRoot(events, values), arrayBound(events, values, fail)!!), nil
}

func arrayBoundValue[T ~[1]int](events *[]string, values %T) int {
	record(events, "bound")
	(*values)[0] = 2
	return 0
}

func genericArrayValueReference[T ~[1]int](events *[]string, values %T) int {
	return consumeArray(events, *arrayRoot(events, values), arrayBoundValue(events, values))
}

func genericArrayValueFailureReference[T ~[1]int](events *[]string, values %T) (int, error) {
	root := arrayRoot(events, values)
	snapshot := *root
	bound, err := arrayBound(events, values, true)
	if err != nil {
		return 0, err
	}
	return consumeArray(events, snapshot, bound), nil
}

type arrayHolder struct {
	Array [1]int
}

func holderRoot(events *[]string, value %arrayHolder) %arrayHolder {
	record(events, "holder")
	return value
}

func holderArraySlice(events *[]string, value %arrayHolder) ([]int, error) {
	result := holderRoot(events, value).Array[arrayBound(events, &value.Array, false)!!:]
	record(events, "after")
	return result, nil
}

func arraySource(events *[]string, values [][1]int) [][1]int {
	record(events, "source")
	return values
}

func arrayIndex(events *[]string) int {
	record(events, "index")
	return 0
}

func indexedArraySlice(events *[]string, values [][1]int) ([]int, error) {
	result := arraySource(events, values)[arrayIndex(events)][arrayBound(events, &values[0], false)!!:]
	record(events, "after")
	return result, nil
}

func arrayBoundOther(events *[]string, fail bool) (int, error) {
	record(events, "bound")
	if fail {
		return 0, errBound
	}
	return 0, nil
}

func arrayBoundOtherValue(events *[]string) int {
	record(events, "bound")
	return 0
}

func nilHolderSlice(events *[]string, fail bool) ([]int, error) {
	var value *arrayHolder
	return holderRoot(events, value).Array[arrayBoundOther(events, fail)!!:], nil
}

func nilHolderSliceReference(events *[]string) []int {
	var value *arrayHolder
	return holderRoot(events, value).Array[arrayBoundOtherValue(events):]
}

func nilHolderSliceFailureReference(events *[]string) ([]int, error) {
	var value *arrayHolder
	root := holderRoot(events, value)
	bound, err := arrayBoundOther(events, true)
	if err != nil {
		return nil, err
	}
	return root.Array[bound:], nil
}

func emptyIndexedArraySlice(events *[]string, fail bool) ([]int, error) {
	var values [][1]int
	return arraySource(events, values)[arrayIndex(events)][arrayBoundOther(events, fail)!!:], nil
}

func emptyIndexedArraySliceReference(events *[]string) []int {
	var values [][1]int
	return arraySource(events, values)[arrayIndex(events)][arrayBoundOtherValue(events):]
}

func emptyIndexedArraySliceFailureReference(events *[]string) ([]int, error) {
	var values [][1]int
	source := arraySource(events, values)
	index := arrayIndex(events)
	bound, err := arrayBoundOther(events, true)
	if err != nil {
		return nil, err
	}
	return source[index][bound:], nil
}
`

const loweringComprehensionSliceIdentityTestSource = `package comprehensionsliceidentity

import (
	"strings"
	"testing"
)

func capturePanic(run func() error) (panicked bool, err error) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	err = run()
	return panicked, err
}

func TestGeneratedComprehensionAndSliceIdentity(t *testing.T) {
	filteredValues := filtered([]int{1, 2, 3})
	if len(filteredValues) != 2 || filteredValues[0] != 1 || filteredValues[1] != 3 {
		t.Fatalf("filtered values=%v", filteredValues)
	}
	nestedValues := nested([][]int{{1, 2}, {3}})
	if len(nestedValues) != 3 || nestedValues[0] != 1 || nestedValues[1] != 2 || nestedValues[2] != 3 {
		t.Fatalf("nested values=%v", nestedValues)
	}

	values := []int{1, 2, 3, 4}
	var copied []int
	allocations := testing.AllocsPerRun(1000, func() {
		copied = identityCopy(values)
	})
	if allocations != 1 || len(copied) != len(values) || &copied[0] == &values[0] {
		t.Fatalf("identity copy allocations=%f values=%v", allocations, copied)
	}
	transformed := transformedCopy(values)
	if len(transformed) != 4 || transformed[0] != 11 || transformed[3] != 14 {
		t.Fatalf("transformed values=%v", transformed)
	}

	events := []string{}
	array := [1]int{1}
	slice, err := genericArraySlice(&events, &array, false)
	if err != nil || len(slice) != 1 || slice[0] != 2 || &slice[0] != &array[0] ||
		strings.Join(events, ",") != "root,bound,after" {
		t.Fatalf("slice=%v array=%v error=%v events=%v", slice, array, err, events)
	}

	events = nil
	array = [1]int{1}
	slice, err = genericArraySlice(&events, &array, true)
	if slice != nil || err != errBound || array[0] != 2 ||
		strings.Join(events, ",") != "root,bound" {
		t.Fatalf("failed slice=%v array=%v error=%v events=%v", slice, array, err, events)
	}

	events = nil
	array = [1]int{1}
	value, err := genericArrayValue(&events, &array, false)
	referenceEvents := []string{}
	referenceArray := [1]int{1}
	reference := genericArrayValueReference(&referenceEvents, &referenceArray)
	if err != nil || value != reference || value != 2 || array[0] != 2 ||
		referenceArray[0] != 2 || strings.Join(events, ",") != "root,bound,consume" ||
		strings.Join(events, ",") != strings.Join(referenceEvents, ",") {
		t.Fatalf("array value=%d reference=%d array=%v reference array=%v "+
			"error=%v events=%v reference events=%v",
			value, reference, array, referenceArray, err, events, referenceEvents)
	}

	events = nil
	array = [1]int{1}
	value, err = genericArrayValue(&events, &array, true)
	referenceEvents = nil
	referenceArray = [1]int{1}
	reference, referenceErr := genericArrayValueFailureReference(&referenceEvents, &referenceArray)
	if err != errBound || referenceErr != errBound || value != reference || value != 0 ||
		array[0] != 2 || referenceArray[0] != 2 || strings.Join(events, ",") != "root,bound" ||
		strings.Join(events, ",") != strings.Join(referenceEvents, ",") {
		t.Fatalf("failed array value=%d reference=%d array=%v reference array=%v "+
			"error=%v reference error=%v events=%v reference events=%v",
			value, reference, array, referenceArray, err, referenceErr,
			events, referenceEvents)
	}

	events = nil
	holder := arrayHolder{Array: [1]int{1}}
	slice, err = holderArraySlice(&events, &holder)
	if err != nil || len(slice) != 1 || slice[0] != 2 || &slice[0] != &holder.Array[0] ||
		strings.Join(events, ",") != "holder,bound,after" {
		t.Fatalf("field slice=%v holder=%v error=%v events=%v", slice, holder, err, events)
	}

	events = nil
	arrays := [][1]int{{1}}
	slice, err = indexedArraySlice(&events, arrays)
	if err != nil || len(slice) != 1 || slice[0] != 2 || &slice[0] != &arrays[0][0] ||
		strings.Join(events, ",") != "source,index,bound,after" {
		t.Fatalf("indexed slice=%v arrays=%v error=%v events=%v", slice, arrays, err, events)
	}

	events = nil
	panicked, err := capturePanic(func() error {
		_, callErr := nilHolderSlice(&events, false)
		return callErr
	})
	referenceEvents = nil
	referencePanicked, referenceErr := capturePanic(func() error {
		_ = nilHolderSliceReference(&referenceEvents)
		return nil
	})
	if panicked != referencePanicked || err != referenceErr ||
		strings.Join(events, ",") != strings.Join(referenceEvents, ",") {
		t.Fatalf("nil field panic=%v reference panic=%v error=%v "+
			"reference error=%v events=%v reference events=%v",
			panicked, referencePanicked, err, referenceErr, events, referenceEvents)
	}

	events = nil
	panicked, err = capturePanic(func() error {
		_, callErr := nilHolderSlice(&events, true)
		return callErr
	})
	referenceEvents = nil
	referencePanicked, referenceErr = capturePanic(func() error {
		_, callErr := nilHolderSliceFailureReference(&referenceEvents)
		return callErr
	})
	if panicked != referencePanicked || panicked || err != errBound || referenceErr != errBound ||
		strings.Join(events, ",") != strings.Join(referenceEvents, ",") {
		t.Fatalf("failed nil field panic=%v reference panic=%v error=%v "+
			"reference error=%v events=%v reference events=%v",
			panicked, referencePanicked, err, referenceErr, events, referenceEvents)
	}

	events = nil
	panicked, err = capturePanic(func() error {
		_, callErr := emptyIndexedArraySlice(&events, false)
		return callErr
	})
	referenceEvents = nil
	referencePanicked, referenceErr = capturePanic(func() error {
		_ = emptyIndexedArraySliceReference(&referenceEvents)
		return nil
	})
	if panicked != referencePanicked || err != referenceErr ||
		strings.Join(events, ",") != strings.Join(referenceEvents, ",") {
		t.Fatalf("empty index panic=%v reference panic=%v error=%v "+
			"reference error=%v events=%v reference events=%v",
			panicked, referencePanicked, err, referenceErr, events, referenceEvents)
	}

	events = nil
	panicked, err = capturePanic(func() error {
		_, callErr := emptyIndexedArraySlice(&events, true)
		return callErr
	})
	referenceEvents = nil
	referencePanicked, referenceErr = capturePanic(func() error {
		_, callErr := emptyIndexedArraySliceFailureReference(&referenceEvents)
		return callErr
	})
	if panicked != referencePanicked || panicked || err != errBound || referenceErr != errBound ||
		strings.Join(events, ",") != strings.Join(referenceEvents, ",") {
		t.Fatalf("failed empty index panic=%v reference panic=%v error=%v "+
			"reference error=%v events=%v reference events=%v",
			panicked, referencePanicked, err, referenceErr, events, referenceEvents)
	}
}
`
