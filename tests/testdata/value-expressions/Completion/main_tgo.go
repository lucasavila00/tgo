package main

import "fmt"

func early(flag bool) int {
	{
		var tgo1 int
		if flag {
			return 7
			_ = "unreachable"
		} else {
			tgo1 = 9
		}
		return tgo1
	}

}

func nested(flag bool) int {
	{
		var tgo2 int
		if flag {
			if !flag {
				return 6
			} else {
				return 8
			}
			_ = "unreachable"
		} else {
			tgo2 = 9
		}
		return tgo2
	}

}

func jump(flag bool) int8 {
	{
		var tgo3 int8
		if flag {
			goto value
			return 0
		value:
			tgo3 = 127
		} else {
			tgo3 = 1
		}
		return tgo3
	}

}

func deferred(flag bool, trace *string) int {
	{
		var tgo4 int
		if flag {
			defer func() { *trace += "D" }()
			return 7
			_ = interface {
			}(nil)
		} else {
			tgo4 = 9
		}
		return tgo4
	}

}

func argument(flag bool) int {
	{
		var tgo5 int
		if flag {
			var tgo6 int
			if !flag {
				return 6
			} else {
				return 8
			}
			var ignored int = tgo6

			_ = ignored
			_ = "unreachable"
		} else {
			tgo5 = 9
		}
		return tgo5
	}

}

func innerBreak(flag bool) int {
	{
		var tgo7 int
		if flag {
			for {
				break
			}
			tgo7 = 3
		} else {
			tgo7 = 9
		}
		return tgo7
	}

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
