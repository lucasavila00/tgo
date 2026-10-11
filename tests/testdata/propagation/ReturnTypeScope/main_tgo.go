package main

import tgo8 "fmt"

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
	type tgo5 = T
	if wrap {
		run := func(T int) (tgo3 T, tgo6 Failure) {
			var tgo4 tgo5 = tgo3
			var tgo7 Failure = tgo6
			{
				var tgo1 int
				var tgo2 Failure
				tgo1, tgo2 = next(fail)
				if tgo2 != nil {
					tgo7 = tgo2
					tgo7 = tgo8.Errorf("%s: %w", "next", tgo7)
					return tgo4, tgo7
				}
				_ = tgo1
			}
			return value, nil
		}
		return run(0)
	}
	run := func(T int) (tgo11 T, tgo13 Failure) {
		var tgo12 tgo5 = tgo11
		var tgo14 Failure = tgo13
		{
			var tgo9 int
			var tgo10 Failure
			tgo9, tgo10 = next(fail)
			if tgo10 != nil {
				tgo14 = tgo10
				return tgo12, tgo14
			}
			_ = tgo9
		}
		return value, nil
	}
	return run(0)
}

func named[T any](value T, fail, wrap bool) (T, Failure) {
	type tgo18 = T
	if wrap {
		run := func() (T T, err Failure) {
			var tgo17 tgo18 = T
			var tgo19 Failure = err
			{
				var tgo15 int
				var tgo16 Failure
				tgo15, tgo16 = next(fail)
				if tgo16 != nil {
					tgo19 = tgo16
					tgo19 = tgo8.Errorf("%s: %w", "next", tgo19)
					return tgo17, tgo19
				}
				_ = tgo15
			}
			return value, nil
		}
		return run()
	}
	run := func() (T T, err Failure) {
		var tgo22 tgo18 = T
		var tgo23 Failure = err
		{
			var tgo20 int
			var tgo21 Failure
			tgo20, tgo21 = next(fail)
			if tgo21 != nil {
				tgo23 = tgo21
				return tgo22, tgo23
			}
			_ = tgo20
		}
		return value, nil
	}
	return run()
}

func annotated(Result int, fail, wrap bool) (tgo26 Visible, tgo28 Failure) {
	var tgo27 Visible = tgo26
	var tgo29 Failure = tgo28
	if wrap {
		{
			var tgo24 int
			var tgo25 Failure
			tgo24, tgo25 = next(fail)
			if tgo25 != nil {
				tgo29 = tgo25
				tgo29 = tgo8.Errorf("%s: %w", "next", tgo29)
				return tgo27, tgo29
			}
			_ = tgo24
		}
	} else {
		{
			var tgo30 int
			var tgo31 Failure
			tgo30, tgo31 = next(fail)
			if tgo31 != nil {
				tgo29 = tgo31
				return tgo27, tgo29
			}
			_ = tgo30
		}
	}
	return Visible{Value: Result}, nil
}

func local(fail, wrap bool) (tgo34 int, tgo36 error) {
	var tgo35 int = tgo34
	var tgo37 error = tgo36
	error := 1
	if wrap {
		{
			var tgo32 int
			var tgo33 Failure
			tgo32, tgo33 = next(fail)
			if tgo33 != nil {
				tgo37 = tgo33
				tgo37 = tgo8.Errorf("%s: %w", "next", tgo37)
				return tgo35, tgo37
			}
			_ = tgo32
		}
	} else {
		{
			var tgo38 int
			var tgo39 Failure
			tgo38, tgo39 = next(fail)
			if tgo39 != nil {
				tgo37 = tgo39
				return tgo35, tgo37
			}
			_ = tgo38
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
