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
	var value int
	switch wrap {
	case true:
		defer fmt.Print("defer ")
		result, err := next(fail)
		if err != nil {
			return 0, fmt.Errorf("next: %w", err)
		}
		value = result
	default:
		defer fmt.Print("defer ")
		result, err := next(fail)
		if err != nil {
			return 0, err
		}
		value = result
	}
	fmt.Print("after ")
	return value, nil
}
func jump(skip bool) int {
	x := 0
	switch skip {
	case true:
		goto done
	default:
		x = 9
	}
done:
	return x
}
func panics() {
	defer func() { fmt.Println("panic", recover() == failure) }()
	before := nextValue()
	var branch int
	switch true {
	case true:
		fmt.Print("branch ")
		panic(failure)
	default:
		branch = 0
	}
	_ = before + branch + laterValue()
}
func nextValue() int  { fmt.Print("before "); return 1 }
func laterValue() int { fmt.Print("later "); return 2 }
func nestedDead() int {
	switch 1 {
	case 1:
		switch 2 {
		case 2:
			return 30
		default:
			return 40
		}
	default:
		return 50
	}
}
func dead(n int) int {
	switch n {
	case 0:
		return 10
	default:
		return 20
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
	fmt.Println(jump(true), jump(false), dead(0), dead(1), nestedDead())
	panics()
}
