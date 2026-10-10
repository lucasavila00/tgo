package compiler

import (
	"strings"
	"testing"
)

func TestLoweringKeepsMapComprehensionValueCalls(t *testing.T) {
	output := compileSourceOutput(t, `package sample

func mapped(value string) string { return "value:" + value }

func build(values []string) map[string]string {
	return map[string]string{for _, value := range values {
		value: mapped(value)
	}}
}
`)
	if !strings.Contains(output, "result[value] = mapped(value)") {
		t.Fatalf("generated map comprehension changed its value call\n%s", output)
	}
	if strings.Contains(output, "append(value)") {
		t.Fatalf("generated map comprehension rewrote a value call as append\n%s", output)
	}
}

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
