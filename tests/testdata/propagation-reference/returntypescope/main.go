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
	var zero T
	if wrap {
		run := func(T int) (T, Failure) {
			_, callErr := next(fail)
			if callErr != nil {
				return zero, fmt.Errorf("next: %w", callErr)
			}
			return value, nil
		}
		return run(0)
	}
	run := func(T int) (T, Failure) {
		_, callErr := next(fail)
		if callErr != nil {
			return zero, callErr
		}
		return value, nil
	}
	return run(0)
}

func named[T any](value T, fail, wrap bool) (T, Failure) {
	var zero T
	if wrap {
		run := func() (T T, err Failure) {
			_, callErr := next(fail)
			if callErr != nil {
				return zero, fmt.Errorf("next: %w", callErr)
			}
			return value, nil
		}
		return run()
	}
	run := func() (T T, err Failure) {
		_, callErr := next(fail)
		if callErr != nil {
			return zero, callErr
		}
		return value, nil
	}
	return run()
}

func annotated(Result int, fail, wrap bool) (Visible, Failure) {
	if wrap {
		_, callErr := next(fail)
		if callErr != nil {
			var zero Visible
			return zero, fmt.Errorf("next: %w", callErr)
		}
	} else {
		_, callErr := next(fail)
		if callErr != nil {
			var zero Visible
			return zero, callErr
		}
	}
	return Visible{Value: Result}, nil
}

func local(fail, wrap bool) (int, error) {
	error := 1
	_, callErr := next(fail)
	if callErr != nil {
		if wrap {
			return 0, fmt.Errorf("next: %w", callErr)
		}
		return 0, callErr
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
