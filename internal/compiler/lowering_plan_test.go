package compiler

import (
	"strings"
	"testing"
)

func TestTypedPlanEmitsGuardedPropagation(t *testing.T) {
	output := compileSourceOutput(t, `package sample

func check() (bool, error) { return true, nil }

func use(ready bool) (bool, error) {
	return ready && check()!!, nil
}
`)
	function := strings.Index(output, "func use")
	guard := strings.Index(output[function:], "\n\tif ")
	call := strings.LastIndex(output, "check()") - function
	errorCheck := strings.Index(output[function:], "if err != nil")
	if guard < 0 || call < guard || errorCheck < call {
		t.Fatalf("generated output does not guard propagation:\n%s", output)
	}
}
