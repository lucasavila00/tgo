package app

import "testing"

func TestNamedResults(t *testing.T) {
	values := make(chan int, 1)
	values <- 4
	close(values)

	tests := []struct {
		name string
		got  int
		want int
	}{
		{name: "loop", got: resultFromLoop(true), want: 1},
		{name: "switch", got: resultFromSwitch(1), want: 2},
		{name: "select", got: resultFromSelect(values), want: 4},
		{name: "capture", got: resultFromLiteral(), want: 1},
		{name: "literal", got: literalResult(), want: 3},
		{name: "write", got: writeInLiteral(), want: 2},
		{name: "return before post", got: returnBeforePost(true), want: 1},
		{name: "skip loop", got: returnBeforePost(false), want: 2},
		{name: "goto", got: resultBeforeGoto(), want: 1},
		{name: "nested continue", got: nestedContinue(true), want: 1},
		{name: "outer continue", got: outerContinue(true), want: 2},
		{name: "outer break", got: outerBreak(true), want: 2},
		{name: "unreachable continue", got: unreachableContinue(true), want: 1},
		{name: "switch fallthrough", got: switchFallthrough(true), want: 1},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s: got %d, want %d", test.name, test.got, test.want)
		}
	}
}
