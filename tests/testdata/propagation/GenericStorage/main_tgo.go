package main

import reflect_tgo2 "reflect"

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("failure")
var visit, failAt int

func step(name string) (int, error) {
	fmt.Print(name)
	visit++
	if visit == failAt {
		return 0, sentinel
	}
	return 1, nil
}

func set[T interface{ ~[2]int | ~[]int }](a, b T) error {
	type tgo1 = T

	index := func() (int, error) {
		a = b
		return step("I")
	}
	{
		T := 3
		_ = T
		var tgo3 bool = reflect_tgo2.TypeFor[tgo1]().Kind() == reflect_tgo2.Array
		var tgo4 tgo1
		if !tgo3 {
			tgo4 = a
		}
		var value_tgo5 int
		var error_tgo6 error
		value_tgo5, error_tgo6 = index()
		if error_tgo6 != nil {
			return error_tgo6
		}
		var value_tgo7 int
		var error_tgo8 error
		value_tgo7, error_tgo8 = step("V")
		if error_tgo8 != nil {
			return error_tgo8
		}
		var value_tgo9 int = value_tgo7
		(*func() *tgo1 {
			if tgo3 {
				return &a
			}
			return &tgo4
		}())[value_tgo5] = value_tgo9
	}
	fmt.Print(a)
	return nil

}

func arrays[T interface{ ~[2]int | ~[3]int }](a, b T) error {
	index := func() (int, error) {
		a = b
		return step("I")
	}
	var value_tgo10 int
	var error_tgo11 error
	value_tgo10, error_tgo11 = index()
	if error_tgo11 != nil {
		return error_tgo11
	}
	var value_tgo12 int
	var error_tgo13 error
	value_tgo12, error_tgo13 = step("V")
	if error_tgo13 != nil {
		return error_tgo13
	}
	a[value_tgo10] = value_tgo12
	fmt.Print(a)
	return nil
}

type Box[T interface{ ~[2]int | ~[]int }] struct{ Value T }

func (b *Box[T]) Set() error {
	type tgo14 = T
	defer func() {
		if recover() != nil {
			fmt.Print("r")
		}
	}()
	var value_tgo15 *Box[T] = b
	var tgo16 bool = reflect_tgo2.TypeFor[tgo14]().Kind() == reflect_tgo2.Array
	var tgo17 tgo14
	if !tgo16 {
		tgo17 = value_tgo15.Value
	}
	var value_tgo18 int
	var error_tgo19 error
	value_tgo18, error_tgo19 = step("I")
	if error_tgo19 != nil {
		return error_tgo19
	}
	var value_tgo20 int
	var error_tgo21 error
	value_tgo20, error_tgo21 = step("V")
	if error_tgo21 != nil {
		return error_tgo21
	}
	var value_tgo22 int = value_tgo20
	(*func() *tgo14 {
		if tgo16 {
			return &value_tgo15.Value
		}
		return &tgo17
	}())[value_tgo18] = value_tgo22
	fmt.Print(b.Value)
	return nil
}

func show(err error) {
	if err == nil {
		fmt.Print("S|")
	} else if err == sentinel {
		fmt.Print("E|")
	} else {
		panic(err)
	}
}

func main() {
	for _, failure := range []int{0, 1, 2} {

		visit = 0
		failAt = failure
		show(set([2]int{}, [2]int{7, 8}))
		visit = 0
		failAt = failure
		a := []int{0, 0}
		show(set(a, []int{7, 8}))
		fmt.Print(a, "|")
		visit = 0
		failAt = failure
		show(arrays([2]int{}, [2]int{7, 8}))
		visit = 0
		failAt = failure
		var b *Box[[2]int]
		show(b.Set())

	}
}
