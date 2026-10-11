package main

import tgo9 "fmt"

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

func calls(fail bool) (tgo3 error) {
	var tgo4 error = tgo3
	var tgo5 error = tgo4
	{
		var tgo6 uint64 = 1 << count()
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = next(fail)
		tgo4 = tgo2
		if tgo4 != tgo5 {
			return tgo4
		}
		use(tgo6, tgo1)
	}
	{
		var tgo10 uint64 = 1 << count()
		var tgo7 int
		var tgo8 error
		tgo7, tgo8 = next(fail)
		tgo4 = tgo8
		if tgo4 != tgo5 {
			tgo4 = tgo9.Errorf("%s: %w", "next", tgo4)
			return tgo4
		}
		use(tgo10, tgo7)
	}
	return nil
}

func returns(fail bool) (tgo13 uint64, tgo15 int, tgo17 error) {
	var tgo14 uint64 = tgo13
	var tgo16 int = tgo15
	var tgo18 error = tgo17
	var tgo19 error = tgo18

	n := uint(40)
	{
		var tgo20 uint64 = 1 << n
		var tgo11 int
		var tgo12 error
		tgo11, tgo12 = next(fail)
		tgo18 = tgo12
		if tgo18 != tgo19 {
			return tgo14, tgo16, tgo18
		}
		return tgo20, tgo11, nil
	}
}

func assign(fail bool) (tgo23 error) {
	var tgo24 error = tgo23
	var tgo25 error = tgo24

	n := uint(40)
	var first uint64
	var second int
	{
		var tgo26 uint64 = 1 << n
		var tgo21 int
		var tgo22 error
		tgo21, tgo22 = next(fail)
		tgo24 = tgo22
		if tgo24 != tgo25 {
			return tgo24
		}
		first, second = tgo26, tgo21
	}
	use(first, second)
	var tgo29 uint64 = 1 << n
	var tgo27 int
	var tgo28 error
	tgo27, tgo28 = next(fail)
	tgo24 = tgo28
	if tgo24 != tgo25 {
		tgo24 = tgo9.Errorf("%s: %w", "next", tgo24)
		return tgo24
	}
	pair := Pair{First: tgo29, Second: tgo27}
	use(pair.First, pair.Second)
	return nil
}

func useNumber(value Number, next int) { fmt.Printf("%d %d ", value, next) }

func named(fail bool) (tgo32 error) {
	var tgo33 error = tgo32
	var tgo34 error = tgo33

	n := uint(40)
	var value Number = 1 << n
	{
		var tgo35 Number = value
		var tgo30 int
		var tgo31 error
		tgo30, tgo31 = next(fail)
		tgo33 = tgo31
		if tgo33 != tgo34 {
			return tgo33
		}
		useNumber(tgo35, tgo30)
	}
	return nil
}

func generic[T ~uint64](value T, fail bool) (tgo38 error) {
	var tgo39 error = tgo38
	var tgo40 error = tgo39

	consume := func(first T, second int) { fmt.Printf("%d %d ", first, second) }
	{
		var tgo41 func(first T, second int) = consume
		var tgo36 int
		var tgo37 error
		tgo36, tgo37 = next(fail)
		tgo39 = tgo37
		if tgo39 != tgo40 {
			return tgo39
		}
		tgo41(1<<uint(40), tgo36)
	}
	{
		var tgo44 func(first T, second int) = consume
		var tgo45 T = value
		var tgo42 int
		var tgo43 error
		tgo42, tgo43 = next(fail)
		tgo39 = tgo43
		if tgo39 != tgo40 {
			return tgo39
		}
		tgo44(tgo45, tgo42)
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
