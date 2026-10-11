package main

import fmt_tgo6 "fmt"

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
	{
		var tgo3 uint64 = 1 << count()
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = next(fail)
		if tgo2 != nil {
			return tgo2
		}
		use(tgo3, tgo1)
	}
	{
		var tgo7 uint64 = 1 << count()
		var tgo4 int
		var tgo5 error
		tgo4, tgo5 = next(fail)
		if tgo5 != nil {
			return fmt_tgo6.Errorf("%s: %w", "next", tgo5)
		}
		use(tgo7, tgo4)
	}
	return nil
}

func returns(fail bool) (uint64, int, error) {
	var tgo10 uint64
	var tgo11 int
	n := uint(40)
	{
		var tgo12 uint64 = 1 << n
		var tgo8 int
		var tgo9 error
		tgo8, tgo9 = next(fail)
		if tgo9 != nil {
			return tgo10, tgo11, tgo9
		}
		return tgo12, tgo8, nil
	}
}

func assign(fail bool) error {
	n := uint(40)
	var first uint64
	var second int
	{
		var tgo15 uint64 = 1 << n
		var tgo13 int
		var tgo14 error
		tgo13, tgo14 = next(fail)
		if tgo14 != nil {
			return tgo14
		}
		first, second = tgo15, tgo13
	}
	use(first, second)
	var tgo18 uint64 = 1 << n
	var tgo16 int
	var tgo17 error
	tgo16, tgo17 = next(fail)
	if tgo17 != nil {
		return fmt_tgo6.Errorf("%s: %w", "next", tgo17)
	}
	pair := Pair{First: tgo18, Second: tgo16}
	use(pair.First, pair.Second)
	return nil
}

func useNumber(value Number, next int) { fmt.Printf("%d %d ", value, next) }

func named(fail bool) error {
	n := uint(40)
	var value Number = 1 << n
	{
		var tgo21 Number = value
		var tgo19 int
		var tgo20 error
		tgo19, tgo20 = next(fail)
		if tgo20 != nil {
			return tgo20
		}
		useNumber(tgo21, tgo19)
	}
	return nil
}

func generic[T ~uint64](value T, fail bool) error {
	consume := func(first T, second int) { fmt.Printf("%d %d ", first, second) }
	{
		var tgo24 func(first T, second int) = consume
		var tgo22 int
		var tgo23 error
		tgo22, tgo23 = next(fail)
		if tgo23 != nil {
			return tgo23
		}
		tgo24(1<<uint(40), tgo22)
	}
	{
		var tgo27 func(first T, second int) = consume
		var tgo28 T = value
		var tgo25 int
		var tgo26 error
		tgo25, tgo26 = next(fail)
		if tgo26 != nil {
			return tgo26
		}
		tgo27(tgo28, tgo25)
	}
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
