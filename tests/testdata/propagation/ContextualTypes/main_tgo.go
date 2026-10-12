package main

import (
	"errors"
	"fmt"
)

type Number uint64
type Pair struct {
	First  uint64
	Second int
}

var sentinel = errors.New("failed")
var trace string

func next(fail bool) (int, error) {
	trace += "N"
	if fail {
		return 0, sentinel
	}
	return 3, nil
}

func use(value uint64, other int) {
	fmt.Printf("%d %d ", value, other)
}

func count() uint { trace += "C"; return 40 }

func calls(fail bool) error {
	var value_tgo3 uint64 = 1 << count()
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = next(fail)
	if error_tgo2 != nil {
		return error_tgo2
	}
	use(value_tgo3, value_tgo1)
	var value_tgo6 uint64 = 1 << count()
	var value_tgo4 int
	var error_tgo5 error
	value_tgo4, error_tgo5 = next(fail)
	if error_tgo5 != nil {
		return fmt.Errorf("next: %w", error_tgo5)
	}
	use(value_tgo6, value_tgo4)
	return nil
}

func returns(fail bool) (uint64, int, error) {
	var zero_tgo9 uint64
	var zero_tgo10 int
	n := uint(40)
	var value_tgo11 uint64 = 1 << n
	var value_tgo7 int
	var error_tgo8 error
	value_tgo7, error_tgo8 = next(fail)
	if error_tgo8 != nil {
		return zero_tgo9, zero_tgo10, error_tgo8
	}
	return value_tgo11, value_tgo7, nil
}

func assign(fail bool) error {
	n := uint(40)
	var first uint64
	var second int
	var value_tgo14 uint64 = 1 << n
	var value_tgo12 int
	var error_tgo13 error
	value_tgo12, error_tgo13 = next(fail)
	if error_tgo13 != nil {
		return error_tgo13
	}
	first, second = value_tgo14, value_tgo12
	use(first, second)
	var value_tgo17 uint64 = 1 << n
	var value_tgo15 int
	var error_tgo16 error
	value_tgo15, error_tgo16 = next(fail)
	if error_tgo16 != nil {
		return fmt.Errorf("next: %w", error_tgo16)
	}
	pair := Pair{First: value_tgo17, Second: value_tgo15}
	use(pair.First, pair.Second)
	return nil
}

func useNumber(value Number, next int) { fmt.Printf("%d %d ", value, next) }

func named(fail bool) error {
	n := uint(40)
	var value Number = 1 << n
	var value_tgo20 Number = value
	var value_tgo18 int
	var error_tgo19 error
	value_tgo18, error_tgo19 = next(fail)
	if error_tgo19 != nil {
		return error_tgo19
	}
	useNumber(value_tgo20, value_tgo18)
	return nil
}

func generic[T ~uint64](value T, fail bool) error {
	consume := func(first T, second int) { fmt.Printf("%d %d ", first, second) }
	var value_tgo23 func(first T, second int) = consume
	var value_tgo21 int
	var error_tgo22 error
	value_tgo21, error_tgo22 = next(fail)
	if error_tgo22 != nil {
		return error_tgo22
	}
	value_tgo23(1<<uint(40), value_tgo21)
	var value_tgo26 func(first T, second int) = consume
	var value_tgo27 T = value
	var value_tgo24 int
	var error_tgo25 error
	value_tgo24, error_tgo25 = next(fail)
	if error_tgo25 != nil {
		return error_tgo25
	}
	value_tgo26(value_tgo27, value_tgo24)
	return nil
}

func main() {
	fmt.Print(calls(false), " ")
	first, second, err := returns(false)
	fmt.Printf("%d %d %v ", first, second, err)
	fmt.Print(assign(false), " ")
	fmt.Print(named(false), " ")
	fmt.Print(generic(Number(9), false), " ")
	fmt.Print(calls(true) == sentinel, " ", trace)
}
