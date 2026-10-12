package main

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
	var zero_tgo4 int
	var tgo1 int
	switch wrap {
	case true:
		defer fmt.Print("defer ")
		var value_tgo2 int
		var error_tgo3 error
		value_tgo2, error_tgo3 = next(fail)
		if error_tgo3 != nil {
			return zero_tgo4, fmt.Errorf("next: %w", error_tgo3)
		}
		tgo1 = value_tgo2
	default:
		defer fmt.Print("defer ")
		var value_tgo5 int
		var error_tgo6 error
		value_tgo5, error_tgo6 = next(fail)
		if error_tgo6 != nil {
			return zero_tgo4, error_tgo6
		}
		tgo1 = value_tgo5
	}
	value := tgo1

	fmt.Print("after ")
	return value, nil
}
func jump(skip bool) int {
	x := 0
	{
		var tgo7 int
		switch skip {
		case true:
			goto done
		default:
			tgo7 = 9
		}
		x = tgo7
	}
done:
	return x
}
func panics() {
	defer func() { fmt.Println("panic", recover() == failure) }()
	var value_tgo9 int = nextValue()
	var tgo8 int
	switch true {
	case true:
		fmt.Print("branch ")
		panic(failure)
	default:
		tgo8 = 0
	}
	_ = value_tgo9 + tgo8 + laterValue()
}
func nextValue() int  { fmt.Print("before "); return 1 }
func laterValue() int { fmt.Print("later "); return 2 }
func nestedDead() int {
	var tgo10 int
	switch 1 {
	case 1:
		switch 2 {
		case 2:
			return 30
			_ = "unreachable" == "unreachable"
		default:
			return 40
		}
	default:
		tgo10 = 50
	}
	return tgo10
}
func dead(n int) int {
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
func main() {
	for _, tc := range []struct{ wrap, fail bool }{{false, false}, {false, true}, {true, true}} {
		value, err := checked(tc.wrap, tc.fail)
		var typed *failureType
		fmt.Println(value, err == failure, errors.Is(err, failure), errors.As(err, &typed), typed == failure)
		if tc.wrap && err.Error() != "next: failure" {
			panic(err)
		}
	}
	fmt.Println(jump(true), jump(false), dead(0), dead(1), nestedDead())
	panics()
}
