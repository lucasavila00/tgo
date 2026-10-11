package main

import tgo8 "fmt"

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
	{
		var tgo5 uint64 = 1 << count()
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = next(fail)
		if tgo2 != nil {
			tgo4 = tgo2
			return tgo4
		}
		use(tgo5, tgo1)
	}
	{
		var tgo9 uint64 = 1 << count()
		var tgo6 int
		var tgo7 error
		tgo6, tgo7 = next(fail)
		if tgo7 != nil {
			tgo4 = tgo7
			tgo4 = tgo8.Errorf("%s: %w", "next", tgo4)
			return tgo4
		}
		use(tgo9, tgo6)
	}
	return nil
}

func returns(fail bool) (tgo12 uint64, tgo14 int, tgo16 error) {
	var tgo13 uint64 = tgo12
	var tgo15 int = tgo14
	var tgo17 error = tgo16

	n := uint(40)
	{
		var tgo18 uint64 = 1 << n
		var tgo10 int
		var tgo11 error
		tgo10, tgo11 = next(fail)
		if tgo11 != nil {
			tgo17 = tgo11
			return tgo13, tgo15, tgo17
		}
		return tgo18, tgo10, nil
	}
}

func assign(fail bool) (tgo21 error) {
	var tgo22 error = tgo21

	n := uint(40)
	var first uint64
	var second int
	{
		var tgo23 uint64 = 1 << n
		var tgo19 int
		var tgo20 error
		tgo19, tgo20 = next(fail)
		if tgo20 != nil {
			tgo22 = tgo20
			return tgo22
		}
		first, second = tgo23, tgo19
	}
	use(first, second)
	var tgo26 uint64 = 1 << n
	var tgo24 int
	var tgo25 error
	tgo24, tgo25 = next(fail)
	if tgo25 != nil {
		tgo22 = tgo25
		tgo22 = tgo8.Errorf("%s: %w", "next", tgo22)
		return tgo22
	}
	pair := Pair{First: tgo26, Second: tgo24}
	use(pair.First, pair.Second)
	return nil
}

func useNumber(value Number, next int) { fmt.Printf("%d %d ", value, next) }

func named(fail bool) (tgo29 error) {
	var tgo30 error = tgo29

	n := uint(40)
	var value Number = 1 << n
	{
		var tgo31 Number = value
		var tgo27 int
		var tgo28 error
		tgo27, tgo28 = next(fail)
		if tgo28 != nil {
			tgo30 = tgo28
			return tgo30
		}
		useNumber(tgo31, tgo27)
	}
	return nil
}

func generic[T ~uint64](value T, fail bool) (tgo34 error) {
	var tgo35 error = tgo34

	consume := func(first T, second int) { fmt.Printf("%d %d ", first, second) }
	{
		var tgo36 func(first T, second int) = consume
		var tgo32 int
		var tgo33 error
		tgo32, tgo33 = next(fail)
		if tgo33 != nil {
			tgo35 = tgo33
			return tgo35
		}
		tgo36(1<<uint(40), tgo32)
	}
	{
		var tgo39 func(first T, second int) = consume
		var tgo40 T = value
		var tgo37 int
		var tgo38 error
		tgo37, tgo38 = next(fail)
		if tgo38 != nil {
			tgo35 = tgo38
			return tgo35
		}
		tgo39(tgo40, tgo37)
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
