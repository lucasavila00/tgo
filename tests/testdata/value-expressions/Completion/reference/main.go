package main

import "fmt"

func early(flag bool) int {
	if flag {
		return 7
	}
	return 9
}

func wide(flag bool) int {
	if flag {
		return 7
	}
	return 9
}

func nested(flag bool) int {
	if flag {
		if !flag {
			return 6
		}
		return 8
	}
	return 9
}

func jump(flag bool) int8 {
	if flag {
		goto value
	value:
		return 127
	}
	return 1
}

func deferred(flag bool, trace *string) int {
	if flag {
		defer func() { *trace += "D" }()
		return 7
	}
	return 9
}

func argument(flag bool) int {
	if flag {
		if !flag {
			return 6
		}
		return 8
	}
	return 9
}

func innerBreak(flag bool) int {
	if flag {
		for {
			break
		}
		return 3
	}
	return 9
}

func allReturn(flag bool) int {
	if flag {
		return 7
	}
	return 9
}
func namedReturn(flag bool) (out int) {
	defer func() { out += 10 }()
	if flag {
		out = 7
		return
	}
	out = 9
	return
}
func pairReturn(flag bool) (int, string) {
	if flag {
		return 7, "a"
	}
	return 9, "b"
}
func genericReturn[T ~int](flag bool) T {
	if flag {
		return T(7)
	}
	return T(9)
}
func loopReturn(flag bool) int {
	if flag {
		return 8
	}
	return 7
}
func main() {
	for _, flag := range []bool{true, false} {
		expect, nestedExpect, jumpExpect, breakExpect := 9, 9, int8(1), 9
		trace, traceExpect := "", ""
		if flag {
			expect, nestedExpect, jumpExpect, breakExpect = 7, 8, 127, 3
			traceExpect = "D"
		}
		if early(flag) != expect || wide(flag) != expect || nested(flag) != nestedExpect || argument(flag) != nestedExpect || jump(flag) != jumpExpect || innerBreak(flag) != breakExpect {
			panic("branch result")
		}
		if allReturn(flag) != expect || namedReturn(flag) != expect+10 || genericReturn[int](flag) != expect {
			panic("function return")
		}
		loopExpect := 7
		if flag {
			loopExpect = 8
		}
		if loopReturn(flag) != loopExpect {
			panic("loop return binding")
		}
		number, text := pairReturn(flag)
		textExpect := "b"
		if flag {
			textExpect = "a"
		}
		if number != expect || text != textExpect {
			panic("multiple returns")
		}
		if deferred(flag, &trace) != expect || trace != traceExpect {
			panic("branch defer")
		}
	}
	fmt.Println("ok")
}
