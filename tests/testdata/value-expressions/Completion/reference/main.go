package main

import "fmt"

func early(flag bool) int {
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

func main() {
	for _, flag := range []bool{true, false} {
		expect, nestedExpect, jumpExpect, breakExpect := 9, 9, int8(1), 9
		trace, traceExpect := "", ""
		if flag {
			expect, nestedExpect, jumpExpect, breakExpect = 7, 8, 127, 3
			traceExpect = "D"
		}
		if early(flag) != expect || nested(flag) != nestedExpect || argument(flag) != nestedExpect || jump(flag) != jumpExpect || innerBreak(flag) != breakExpect {
			panic("branch result")
		}
		if deferred(flag, &trace) != expect || trace != traceExpect {
			panic("branch defer")
		}
	}
	fmt.Println("ok")
}
