package main

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
	var zero_tgo7 T
	value := func() (T, error) {
		if innerFails {
			return initial, failure
		}
		return initial, nil
	}
	inner := func() (int, error) {
		var zero_tgo2 int
		T := 1
		var error_tgo1 error
		_, error_tgo1 = value()
		if error_tgo1 != nil {
			return zero_tgo2, error_tgo1
		}
		{
			var tgo5 bool = 0 == 0
		Loop:
			for i := 0; ; {
				if tgo5 {
					tgo5 = 0 != 0
				} else {
					var value_tgo3 int
					var error_tgo4 error
					value_tgo3, error_tgo4 = advance(i)
					if error_tgo4 != nil {
						return zero_tgo2, error_tgo4
					}
					i = value_tgo3
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
		var tgo10 bool = 0 == 0
	Loop:
		for i := 0; ; {
			if tgo10 {
				tgo10 = 0 != 0
			} else {
				var value_tgo8 int
				var error_tgo9 error
				value_tgo8, error_tgo9 = advance(i)
				if error_tgo9 != nil {
					return zero_tgo7, error_tgo9
				}
				i = value_tgo8
			}
			if !(i < 2) {
				break
			}
			{
				var error_tgo6 error
				_, error_tgo6 = inner()
				if error_tgo6 != nil {
					return zero_tgo7, error_tgo6
				}
				if i == 0 {
					continue Loop
				}
			}
		}
	}
	var error_tgo11 error = fail()
	if error_tgo11 != nil {
		return zero_tgo7, fmt.Errorf("fail: %w", error_tgo11)
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
