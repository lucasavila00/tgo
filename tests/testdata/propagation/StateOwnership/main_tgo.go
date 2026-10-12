package main

import fmt_tgo9 "fmt"

import (
	"errors"
	"fmt"
)

var failure = errors.New("failure")

func fail() error {
	return failure
}

func outer[T any](initial T, innerFails bool) (T, error) {
	type tgo3 = T
	var tgo7 T
	value := func() (T, error) {
		if innerFails {
			return initial, failure
		}
		return initial, nil
	}
	inner := func() (int, error) {
		var tgo4 int
		T := 1
		{
			var tgo1 tgo3
			var tgo2 error
			tgo1, tgo2 = value()
			if tgo2 != nil {
				return tgo4, tgo2
			}
			_ = tgo1
		}
	Loop:
		for i := 0; i < 2; i++ {
			if i == 0 {
				continue Loop
			}
		}
		return T, nil
	}
Loop:
	for i := 0; i < 2; i++ {
		{
			var tgo5 int
			var tgo6 error
			tgo5, tgo6 = inner()
			if tgo6 != nil {
				return tgo7, tgo6
			}
			_ = tgo5
		}
		if i == 0 {
			continue Loop
		}
	}
	{
		var tgo8 error = fail()
		if tgo8 != nil {
			return tgo7, fmt_tgo9.Errorf("fail: %w", tgo8)
		}
	}
	return initial, nil
}

func main() {
	value, err := outer[int](9, false)
	fmt.Println(value, errors.Is(err, failure))
	text, err := outer[string]("unused", true)
	fmt.Printf("%q %t\n", text, err == failure)
	fmt.Println(errors.Is(other(), failure))
}
