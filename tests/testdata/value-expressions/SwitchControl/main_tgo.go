package main

import fmt_tgo5 "fmt"

import (
	"errors"
	"fmt"
)

type failureType struct{}

func (*failureType) Error() string { return "failure" }

var failure = &failureType{}

func next(fail bool) (int, error) {
	fmt.Print("call ")
	if fail {
		return 0, failure
	}
	return 7, nil
}
func checked(wrap, fail bool) (int, error) {
	var tgo4 int
	var tgo1 int
	switch wrap {
	case true:
		defer fmt.Print("defer ")
		var tgo2 int
		var tgo3 error
		tgo2, tgo3 = next(fail)
		if tgo3 != nil {
			return tgo4, fmt_tgo5.Errorf("next: %w", tgo3)
		}
		tgo1 = tgo2
	default:
		defer fmt.Print("defer ")
		var tgo6 int
		var tgo7 error
		tgo6, tgo7 = next(fail)
		if tgo7 != nil {
			return tgo4, tgo7
		}
		tgo1 = tgo6
	}
	value := tgo1

	fmt.Print("after ")
	return value, nil
}
func jump(skip bool) int {
	x := 0
	{
		var tgo8 int
		switch skip {
		case true:
			goto done
		default:
			tgo8 = 9
		}
		x = tgo8
	}
done:
	return x
}
func panics() {
	defer func() { fmt.Println("panic", recover() == failure) }()
	{
		var tgo10 int = nextValue()
		var tgo9 int
		switch true {
		case true:
			fmt.Print("branch ")
			panic(failure)
		default:
			tgo9 = 0
		}
		_ = tgo10 + tgo9 + laterValue()
	}
}
func nextValue() int  { fmt.Print("before "); return 1 }
func laterValue() int { fmt.Print("later "); return 2 }
func dead(n int) int {
	{
		var tgo11 int
		switch n {
		case 0:
			return 10
			_ = "unreachable" == "unreachable"
		default:
			tgo11 = 20
		}
		return tgo11
	}
}
func main() {
	for _, tc := range []struct{ wrap, fail bool }{{false, false}, {false, true}, {true, true}} {
		value, err := checked(tc.wrap, tc.fail)
		var typed *failureType
		fmt.Println(value, err == failure, errors.Is(err, failure), errors.As(err, &typed), typed == failure)
		if tc.wrap && err.Error() != "next: failure" {
			panic(err)
		}
	}
	fmt.Println(jump(true), jump(false), dead(0), dead(1))
	panics()
}
