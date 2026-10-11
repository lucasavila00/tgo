package main

import fmt_tgo5 "fmt"

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
	type tgo4 = T
	if wrap {
		run := func(T int) (T, Failure) {
			var tgo3 tgo4
			{
				var tgo1 int
				var tgo2 Failure
				tgo1, tgo2 = next(fail)
				if tgo2 != nil {
					return tgo3, fmt_tgo5.Errorf("next: %w", tgo2)
				}
				_ = tgo1
			}
			return value, nil
		}
		return run(0)
	}
	run := func(T int) (T, Failure) {
		var tgo8 tgo4
		{
			var tgo6 int
			var tgo7 Failure
			tgo6, tgo7 = next(fail)
			if tgo7 != nil {
				return tgo8, tgo7
			}
			_ = tgo6
		}
		return value, nil
	}
	return run(0)
}

func named[T any](value T, fail, wrap bool) (T, Failure) {
	type tgo12 = T
	if wrap {
		run := func() (T T, err Failure) {
			var tgo11 tgo12
			{
				var tgo9 int
				var tgo10 Failure
				tgo9, tgo10 = next(fail)
				if tgo10 != nil {
					return tgo11, fmt_tgo5.Errorf("next: %w", tgo10)
				}
				_ = tgo9
			}
			return value, nil
		}
		return run()
	}
	run := func() (T T, err Failure) {
		var tgo15 tgo12
		{
			var tgo13 int
			var tgo14 Failure
			tgo13, tgo14 = next(fail)
			if tgo14 != nil {
				return tgo15, tgo14
			}
			_ = tgo13
		}
		return value, nil
	}
	return run()
}

func annotated(Result int, fail, wrap bool) (Visible, Failure) {
	var tgo18 Visible
	if wrap {
		{
			var tgo16 int
			var tgo17 Failure
			tgo16, tgo17 = next(fail)
			if tgo17 != nil {
				return tgo18, fmt_tgo5.Errorf("next: %w", tgo17)
			}
			_ = tgo16
		}
	} else {
		{
			var tgo19 int
			var tgo20 Failure
			tgo19, tgo20 = next(fail)
			if tgo20 != nil {
				return tgo18, tgo20
			}
			_ = tgo19
		}
	}
	return Visible{Value: Result}, nil
}

func local(fail, wrap bool) (int, error) {
	var tgo23 int
	error := 1
	if wrap {
		{
			var tgo21 int
			var tgo22 Failure
			tgo21, tgo22 = next(fail)
			if tgo22 != nil {
				return tgo23, fmt_tgo5.Errorf("next: %w", tgo22)
			}
			_ = tgo21
		}
	} else {
		{
			var tgo24 int
			var tgo25 Failure
			tgo24, tgo25 = next(fail)
			if tgo25 != nil {
				return tgo23, tgo25
			}
			_ = tgo24
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
