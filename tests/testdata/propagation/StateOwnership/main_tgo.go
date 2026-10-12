package main

import fmt_tgo15 "fmt"

import (
	"errors"
	"fmt"
)

var failure = errors.New("failure")

func fail() error {
	return failure
}

func advance(i int) (int, error) {
	return i + 1, nil
}

func outer[T any](initial T, innerFails bool) (T, error) {
	type tgo3 = T
	var tgo10 T
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
		{
			var tgo7 bool = 0 == 0
		Loop:
			for i := 0; ; {
				if tgo7 {
					tgo7 = 0 != 0
				} else {
					var tgo5 int
					var tgo6 error
					tgo5, tgo6 = advance(i)
					if tgo6 != nil {
						return tgo4, tgo6
					}
					i = tgo5
				}
				if !(i < 2) {
					break
				}
				{
					if i == 0 {
						continue Loop
					}
				}
			}
		}
		return T, nil
	}
	{
		var tgo13 bool = 0 == 0
	Loop:
		for i := 0; ; {
			if tgo13 {
				tgo13 = 0 != 0
			} else {
				var tgo11 int
				var tgo12 error
				tgo11, tgo12 = advance(i)
				if tgo12 != nil {
					return tgo10, tgo12
				}
				i = tgo11
			}
			if !(i < 2) {
				break
			}
			{
				{
					var tgo8 int
					var tgo9 error
					tgo8, tgo9 = inner()
					if tgo9 != nil {
						return tgo10, tgo9
					}
					_ = tgo8
				}
				if i == 0 {
					continue Loop
				}
			}
		}
	}
	{
		var tgo14 error = fail()
		if tgo14 != nil {
			return tgo10, fmt_tgo15.Errorf("fail: %w", tgo14)
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
