package main

import "fmt"

func early(flag bool) int {
	{
		var tgo1 int
		if flag {
			return 7
			_ = "unreachable" == "unreachable"
		} else {
			tgo1 = 9
		}
		return tgo1
	}

}

func wide(flag bool) int {
	{
		var tgo2 int
		if flag {
			return 7
			_ = 1<<100 == 1<<100
		} else {
			tgo2 = 9
		}
		return tgo2
	}

}

func nested(flag bool) int {
	{
		var tgo3 int
		if flag {
			if !flag {
				return 6
			} else {
				return 8
			}
			_ = "unreachable" == "unreachable"
		} else {
			tgo3 = 9
		}
		return tgo3
	}

}

func jump(flag bool) int8 {
	{
		var tgo4 int8
		if flag {
			goto value
			return 0
		value:
			tgo4 = 127
		} else {
			tgo4 = 1
		}
		return tgo4
	}

}

func deferred(flag bool, trace *string) int {
	{
		var tgo5 int
		if flag {
			defer func() { *trace += "D" }()
			return 7
			_ = interface {
			}(nil)
		} else {
			tgo5 = 9
		}
		return tgo5
	}

}

func argument(flag bool) int {
	{
		var tgo6 int
		if flag {
			var tgo7 int
			if !flag {
				return 6
			} else {
				return 8
			}
			var ignored int = tgo7

			_ = ignored
			_ = "unreachable" == "unreachable"
		} else {
			tgo6 = 9
		}
		return tgo6
	}

}

func innerBreak(flag bool) int {
	{
		var tgo8 int
		if flag {
			for {
				break
			}
			tgo8 = 3
		} else {
			tgo8 = 9
		}
		return tgo8
	}

}

func allReturn(flag bool) int {
	var tgo11 int
	{
		var tgo9 int
		if flag {
			{
				tgo11 = 7
				goto tgo10
			}
		} else {
			{
				tgo11 = 9
				goto tgo10
			}
		}
		var ignored int = tgo9

		_ = ignored
	}
tgo10:
	return tgo11
}

func namedReturn(flag bool) (out int) {
	var tgo14 int
	{
		defer func() { out += 10 }()
		var tgo12 int
		if flag {
			out = 7
			{
				tgo14 = out
				goto tgo13
			}

		} else {
			out = 9
			{
				tgo14 = out
				goto tgo13
			}

		}
		var ignored int = tgo12

		_ = ignored
	}
tgo13:
	return tgo14
}

func pairReturn(flag bool) (int, string) {
	var tgo17 int
	var tgo18 string
	{
		var tgo15 int
		if flag {
			{
				tgo17, tgo18 = 7, "a"
				goto tgo16
			}
		} else {
			{
				tgo17, tgo18 = 9, "b"
				goto tgo16
			}
		}
		var ignored int = tgo15

		_ = ignored
	}
tgo16:
	return tgo17, tgo18
}

func genericReturn[T ~int](flag bool) T {
	var tgo21 T
	{
		var tgo19 int
		if flag {
			{
				tgo21 = T(7)
				goto tgo20
			}
		} else {
			{
				tgo21 = T(9)
				goto tgo20
			}
		}
		var ignored int = tgo19

		_ = ignored
	}
tgo20:
	return tgo21
}

func loopReturn(flag bool) int {
	var tgo29 int
	{
		i := 100
		_ = i
		{
			var tgo25 int
			var tgo26 bool = 0 == 0
			for i_tgo24 := tgo25; ; {
				if tgo26 {
					tgo26 = 0 != 0
					var tgo23 int
					if flag {
						continue
					} else {
						tgo23 = 0
					}
					i_tgo24 = tgo23
				} else {
					i_tgo24++
				}
				if !(i_tgo24 < 2) {
					break
				}
				{
					capture := func() int { return i_tgo24 }
					var tgo22 int
					if flag {
						{
							tgo29 = capture() + 7
							goto tgo28
						}
					} else {
						{
							tgo29 = capture() + 7
							goto tgo28
						}
					}
					var ignored int = tgo22

					_ = ignored
				}
			}
		}
		var tgo27 int
		if flag {
			{
				tgo29 = 9
				goto tgo28
			}
		} else {
			{
				tgo29 = 9
				goto tgo28
			}
		}
		var ignored int = tgo27

		_ = ignored
	}
tgo28:
	return tgo29
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
