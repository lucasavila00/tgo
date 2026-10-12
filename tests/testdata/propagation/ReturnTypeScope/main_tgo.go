package main

import (
	"errors"
	"fmt"
)

type Failure interface{ Error() string }
type Result struct{ Value int }
type Visible = Result

var sentinel = errors.New("failed")

func next(fail bool) (int, Failure) {
	if fail {
		return 0, sentinel
	}
	return 7, nil
}

func generic[T any](value T, fail, wrap bool) (T, Failure) {
	type tgo3 = T
	if wrap {
		run := func(T int) (T, Failure) {
			var zero_tgo2 tgo3
			var error_tgo1 Failure
			_, error_tgo1 = next(fail)
			if error_tgo1 != nil {
				return zero_tgo2, fmt.Errorf("next: %w", error_tgo1)
			}
			return value, nil
		}
		return run(0)
	}
	run := func(T int) (T, Failure) {
		var zero_tgo5 tgo3
		var error_tgo4 Failure
		_, error_tgo4 = next(fail)
		if error_tgo4 != nil {
			return zero_tgo5, error_tgo4
		}
		return value, nil
	}
	return run(0)
}

func named[T any](value T, fail, wrap bool) (T, Failure) {
	type tgo8 = T
	if wrap {
		run := func() (T T, err Failure) {
			var zero_tgo7 tgo8
			var error_tgo6 Failure
			_, error_tgo6 = next(fail)
			if error_tgo6 != nil {
				return zero_tgo7, fmt.Errorf("next: %w", error_tgo6)
			}
			return value, nil
		}
		return run()
	}
	run := func() (T T, err Failure) {
		var zero_tgo10 tgo8
		var error_tgo9 Failure
		_, error_tgo9 = next(fail)
		if error_tgo9 != nil {
			return zero_tgo10, error_tgo9
		}
		return value, nil
	}
	return run()
}

func annotated(Result int, fail, wrap bool) (Visible, Failure) {
	var zero_tgo12 Visible
	if wrap {
		var error_tgo11 Failure
		_, error_tgo11 = next(fail)
		if error_tgo11 != nil {
			return zero_tgo12, fmt.Errorf("next: %w", error_tgo11)
		}
	} else {
		var error_tgo13 Failure
		_, error_tgo13 = next(fail)
		if error_tgo13 != nil {
			return zero_tgo12, error_tgo13
		}
	}
	return Visible{Value: Result}, nil
}

func local(fail, wrap bool) (int, error) {
	var zero_tgo15 int
	error := 1
	if wrap {
		var error_tgo14 Failure
		_, error_tgo14 = next(fail)
		if error_tgo14 != nil {
			return zero_tgo15, fmt.Errorf("next: %w", error_tgo14)
		}
	} else {
		var error_tgo16 Failure
		_, error_tgo16 = next(fail)
		if error_tgo16 != nil {
			return zero_tgo15, error_tgo16
		}
	}
	return error, nil
}

func main() {
	for _, fail := range []bool{false, true} {
		for _, wrap := range []bool{false, true} {
			localValue, localError := local(fail, wrap)
			if (!fail && localValue != 1) || (fail && localValue != 0) || errors.Is(localError, sentinel) != fail {
				panic("local type-name binding")
			}
			value, err := generic(7, fail, wrap)
			fmt.Println("parameter", fail, wrap, value, errors.Is(err, sentinel), err == sentinel)
			value, err = named(7, fail, wrap)
			fmt.Println("result", fail, wrap, value, errors.Is(err, sentinel), err == sentinel)
			result, err := annotated(7, fail, wrap)
			fmt.Println("alias", fail, wrap, result.Value, errors.Is(err, sentinel), err == sentinel)
		}
	}
}
