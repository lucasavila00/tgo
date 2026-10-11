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
		{
			var tgo3 bool = reflect_tgo2.TypeFor[tgo1]().Kind() == reflect_tgo2.Array
			var tgo4 tgo1
			if !tgo3 {
				tgo4 = a
			}
			var tgo5 int
			var tgo6 error
			tgo5, tgo6 = index()
			if tgo6 != nil {
				return tgo6
			}
			var tgo7 int
			var tgo8 error
			tgo7, tgo8 = step("V")
			if tgo8 != nil {
				return tgo8
			}
			var tgo9 int = tgo7
			(*func() *tgo1 {
				if tgo3 {
					return &a
				}
				return &tgo4
			}())[tgo5] = tgo9
		}
	}
	fmt.Print(a)
	return nil

}

func arrays[T interface{ ~[2]int | ~[3]int }](a, b T) error {
	index := func() (int, error) {
		a = b
		return step("I")
	}
	{
		var tgo10 int
		var tgo11 error
		tgo10, tgo11 = index()
		if tgo11 != nil {
			return tgo11
		}
		var tgo12 int
		var tgo13 error
		tgo12, tgo13 = step("V")
		if tgo13 != nil {
			return tgo13
		}
		a[tgo10] = tgo12
	}
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
	{
		var tgo15 *Box[T] = b
		var tgo16 bool = reflect_tgo2.TypeFor[tgo14]().Kind() == reflect_tgo2.Array
		var tgo17 tgo14
		if !tgo16 {
			tgo17 = tgo15.Value
		}
		var tgo18 int
		var tgo19 error
		tgo18, tgo19 = step("I")
		if tgo19 != nil {
			return tgo19
		}
		var tgo20 int
		var tgo21 error
		tgo20, tgo21 = step("V")
		if tgo21 != nil {
			return tgo21
		}
		var tgo22 int = tgo20
		(*func() *tgo14 {
			if tgo16 {
				return &tgo15.Value
			}
			return &tgo17
		}())[tgo18] = tgo22
	}
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
