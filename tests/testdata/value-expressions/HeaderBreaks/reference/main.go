package main

import "fmt"

func switchInit(skip bool) string {
	trace := "I"
	if !skip {
		switch value := 1; value {
		case 1:
			trace += "B"
		}
	}
	return trace + "E"
}

func switchTag(skip bool) string {
	trace := "T"
	if !skip {
	choice:
		switch 1 {
		case 1:
			trace += "B"
			break choice
		}
	}
	return trace + "E"
}

func switchCase(skip bool) string {
	trace := "C"
	if !skip {
		switch 1 {
		default:
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
	return trace + "E"
}

func typeGuard(skip bool) string {
	trace := "T"
	if !skip {
		switch value := any(1).(type) {
		case int:
			trace += fmt.Sprint(value)
		}
	}
	return trace + "E"
}

func selectSend(skip bool) string {
	trace := "C"
	channel := make(chan int, 1)
	if !skip {
		trace += "V"
	choice:
		select {
		case channel <- 1:
			trace += "B"
			break choice
		default:
			panic("blocked send")
		}
	}
	return trace + fmt.Sprint(len(channel))
}

func selectStore(skip bool) []int {
	channel := make(chan int, 1)
	channel <- 5
	values := []int{0, 0}
choice:
	select {
	case value := <-channel:
		if skip {
			break choice
		}
		values[0] = value
		values[1]++
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
		if i == 0 {
			continue
		}
		select {
		case <-channel:
			total++
		default:
			total += 10
		}
	}
	return total
}

func restart() int {
	attempts := 0
again:
	attempts++
	if attempts == 1 {
		goto again
	}
	if attempts != 2 {
		panic("switch body")
	}
	return attempts
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
		values := selectStore(skip)
		if skip && (values[0] != 0 || values[1] != 0) || !skip && (values[0] != 5 || values[1] != 1) {
			panic("selected store break")
		}
	}
	if continueReceive() != 11 || restart() != 2 {
		panic("other jump targets")
	}
	fmt.Println("ok")
}
