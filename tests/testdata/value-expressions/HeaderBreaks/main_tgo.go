package main

import "fmt"

func switchInit(skip bool) string {
	trace := ""
	{
		{
			var tgo2 int

			if skip {
				trace += "I"
				goto tgo1

			} else {
				trace += "I"
				tgo2 = 1
			}
			value := tgo2
			switch value {
			case 1:
				trace += "B"
			}
		}
	tgo1:
	}

	return trace + "E"
}

func switchTag(skip bool) string {
	trace := ""
	{
		{
			var tgo4 int

			if skip {
				trace += "T"
				goto tgo3

			} else {
				trace += "T"
				tgo4 = 1
			}
		choice:
			switch tgo4 {

			case 1:
				trace += "B"
				break choice
			}
		}
	tgo3:
	}

	return trace + "E"
}

func switchCase(skip bool) string {
	trace := ""
	{
		{
			var tgo6 int = -1
			if tgo6 == -1 {
				var tgo7 int

				if skip {
					trace += "C"
					goto tgo5

				} else {
					trace += "C"
					tgo7 = 1
				}
				if 1 == tgo7 {
					tgo6 = 1
				}
			}
			if tgo6 == -1 {
				if 1 ==

					2 {
					tgo6 = 2
				}
			}
			if tgo6 == -1 {
				var tgo8 int

				if true {
					trace += "L"
					tgo8 = 3
				} else {
					tgo8 = 4
				}
				if 1 == tgo8 {
					tgo6 = 3
				}
			}
			if tgo6 == -1 {
				tgo6 = 0
			}
			switch tgo6 {
			case 0:
				panic("default case")
			case 1:
				trace += "B"
				fallthrough
			case 2:
				trace += "F"
			case 3:
				panic("later case")
			}
		}
	tgo5:
	}

	return trace + "E"
}

func typeGuard(skip bool) string {
	trace := ""
	{
		{
			var tgo10 any

			if skip {
				trace += "T"
				goto tgo9

			} else {
				trace += "T"
				tgo10 = any(1)
			}
			switch value := (tgo10).(type) {
			case int:
				trace += fmt.Sprint(value)
			}
		}
	tgo9:
	}

	return trace + "E"
}

func selectSend(skip bool) string {
	trace := ""
	channel := make(chan int, 1)
	{
		{
			var tgo12 chan int

			if skip {
				trace += "C"
				goto tgo11

			} else {
				trace += "C"
				tgo12 = channel
			}
			var tgo13 chan int = (tgo12)
			var tgo14 int
			if true {
				trace += "V"
				tgo14 = 1
			} else {
				tgo14 = 2
			}
		choice:
			select {
			case tgo13 <- tgo14:
				trace += "B"
				break choice
			default:
				panic("blocked send")
			}
		}
	tgo11:
	}

	return trace + fmt.Sprint(len(channel))
}

func selectStore(skip bool) []int {
	channel := make(chan int, 1)
	channel <- 5
	values := []int{0, 0}
	{
		{
			var tgo16 chan int = channel
			var tgo17 int
			select {
			case tgo17 = <-tgo16:
				var tgo18 []int = values
				var tgo19 int
				if skip {
					goto tgo15
				} else {
					tgo19 = 0
				}
				tgo18[tgo19] = tgo17
				values[1]++
			}
		}
	tgo15:
	}

	if len(channel) != 0 {
		panic("receive order")
	}
	return values
}

func continueReceive() int {
	channel := make(chan int, 1)
	channel <- 5
	total := 0
	for i := 0; i < 3; i++ {
		{
			var tgo20 chan int

			if i == 0 {
				continue
			} else {
				tgo20 = channel
			}
			select {
			case <-tgo20:
				total++
			default:
				total += 10
			}
		}

	}
	return total
}

func restart() int {
	attempts := 0
again:
	{
		{
			{
				var tgo23 int
				if attempts++; attempts == 1 {
					goto again
				} else if attempts == 2 {
					goto tgo22

				} else {
					tgo23 = 1
				}
				switch tgo23 {

				case 1:
					panic("switch body")
				}
			}
		tgo22:
		}
	}

	return attempts
}

func rangeBreak(skip bool) string {
	trace := ""
	{
		{
			var tgo25 []int

			if skip {
				trace += "C"
				goto tgo24

			} else {
				trace += "C"
				tgo25 = []int{1, 2}
			}
		choice:
			for _, value := range tgo25 {
				trace += fmt.Sprint(value)
				if value == 2 {
					break choice
				}
			}
		}
	tgo24:
	}
	return trace + "E"
}

func rangeBareBreak(skip bool) string {
	trace := ""
	{
		{
			var tgo27 []int
			if skip {
				trace += "C"
				goto tgo26

			} else {
				trace += "C"
				tgo27 = []int{1, 2}
			}
			for range tgo27 {
				trace += "B"
				break
			}
		}
	tgo26:
	}
	return trace + "E"
}

func rangeOuterContinue() int {
	total := 0
outer:
	for i := 0; i < 3; i++ {
		{
			var tgo28 []int
			if i == 0 {
				continue outer
			} else {
				tgo28 = []int{2}
			}
			for _, value := range tgo28 {
				total += value
			}
		}
	}
	return total
}

func main() {
	for _, skip := range []bool{false, true} {
		wantInit, wantTag, wantCase, wantType, wantSend := "IBE", "TBE", "CBFE", "T1E", "CVB1"
		if skip {
			wantInit, wantTag, wantCase, wantType, wantSend = "IE", "TE", "CE", "TE", "C0"
		}
		if switchInit(skip) != wantInit || switchTag(skip) != wantTag || switchCase(skip) != wantCase || typeGuard(skip) != wantType || selectSend(skip) != wantSend {
			panic("header break")
		}
		wantRange, wantBareRange := "C12E", "CBE"
		if skip {
			wantRange, wantBareRange = "CE", "CE"
		}
		if rangeBreak(skip) != wantRange || rangeBareBreak(skip) != wantBareRange {
			panic("range collection break")
		}
		values := selectStore(skip)
		if skip && (values[0] != 0 || values[1] != 0) || !skip && (values[0] != 5 || values[1] != 1) {
			panic("selected store break")
		}
	}
	if continueReceive() != 11 || restart() != 2 || rangeOuterContinue() != 4 {
		panic("other jump targets")
	}
	fmt.Println("ok")
}
