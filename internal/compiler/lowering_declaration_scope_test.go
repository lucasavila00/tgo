package compiler

import "testing"

func TestLoweringCompletesRHSBeforeShortDeclaration(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "declarationscope", sourceName: "declaration_scope.tgo",
		source: loweringDeclarationScopeSource, testName: "declaration_scope_test.tgo",
		testSource:  loweringDeclarationScopeTestSource,
		testPattern: "TestGeneratedDeclarationScope",
	})
}

const loweringDeclarationScopeSource = `package declarationscope

import "errors"

var errLoad = errors.New("load failure")

func first(events *[]string, fail bool) (int, error) {
	*events = append(*events, "first")
	if fail {
		return 0, errLoad
	}
	return 20, nil
}

func use(events *[]string, value int) int {
	*events = append(*events, "use")
	return value + 1
}

func evaluate(events *[]string, fail bool) (int, int, error) {
	x := 10
	{
		x, y := first(events, fail)!!, use(events, x)
		*events = append(*events, "body")
		return x, y, nil
	}
}
`

const loweringDeclarationScopeTestSource = `package declarationscope

import (
	"errors"
	"strings"
	"testing"
)

func TestGeneratedDeclarationScope(t *testing.T) {
	var events []string
	x, y, err := evaluate(&events, false)
	if x != 20 || y != 11 || err != nil || strings.Join(events, ",") != "first,use,body" {
		t.Fatalf("success x=%d y=%d error=%v events=%v", x, y, err, events)
	}
	events = nil
	x, y, err = evaluate(&events, true)
	if x != 0 || y != 0 || !errors.Is(err, errLoad) || strings.Join(events, ",") != "first" {
		t.Fatalf("failure x=%d y=%d error=%v events=%v", x, y, err, events)
	}
}
`
