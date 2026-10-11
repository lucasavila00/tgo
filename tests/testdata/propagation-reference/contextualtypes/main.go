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

func use(value uint64, other int) { fmt.Printf("%d %d ", value, other) }

func count() uint { trace += "C"; return 40 }

func calls(fail bool) error {
	var left uint64 = 1 << count()
	value, err := next(fail)
	if err != nil {
		return err
	}
	use(left, value)
	left = 1 << count()
	value, err = next(fail)
	if err != nil {
		return fmt.Errorf("next: %w", err)
	}
	use(left, value)
	return nil
}

func returns(fail bool) (uint64, int, error) {
	n := uint(40)
	var left uint64 = 1 << n
	value, err := next(fail)
	if err != nil {
		return 0, 0, err
	}
	return left, value, nil
}

func assign(fail bool) error {
	n := uint(40)
	var left uint64 = 1 << n
	value, err := next(fail)
	if err != nil {
		return err
	}
	first, second := left, value
	use(first, second)
	left = 1 << n
	value, err = next(fail)
	if err != nil {
		return fmt.Errorf("next: %w", err)
	}
	pair := Pair{First: left, Second: value}
	use(pair.First, pair.Second)
	return nil
}

func named(fail bool) error {
	n := uint(40)
	var value Number = 1 << n
	other, err := next(fail)
	if err != nil {
		return err
	}
	fmt.Printf("%d %d ", value, other)
	return nil
}

func generic[T ~uint64](value T, fail bool) error {
	other, err := next(fail)
	if err != nil {
		return err
	}
	fmt.Printf("%d %d ", T(1<<uint(40)), other)
	other, err = next(fail)
	if err != nil {
		return err
	}
	fmt.Printf("%d %d ", value, other)
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
