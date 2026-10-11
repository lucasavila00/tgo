package main

import tgo2 "reflect"

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

func set[T interface{ ~[2]int | ~[]int }](a, b T) (tgo7 error) {
	type tgo1 = T
	var tgo8 error = tgo7

	index := func() (int, error) {
		a = b
		return step("I")
	}
	{
		T := 3
		_ = T
		{
			var tgo3 bool = tgo2.TypeFor[tgo1]().Kind() == tgo2.Array
			var tgo4 tgo1
			if !tgo3 {
				tgo4 = a
			}
			var tgo5 int
			var tgo6 error
			tgo5, tgo6 = index()
			if tgo6 != nil {
				tgo8 = tgo6
				return tgo8
			}
			var tgo9 int
			var tgo10 error
			tgo9, tgo10 = step("V")
			if tgo10 != nil {
				tgo8 = tgo10
				return tgo8
			}
			var tgo11 int = tgo9
			(*func() *tgo1 {
				if tgo3 {
					return &a
				}
				return &tgo4
			}())[tgo5] = tgo11
		}
	}
	fmt.Print(a)
	return nil

}

func arrays[T interface{ ~[2]int | ~[3]int }](a, b T) (tgo14 error) {
	var tgo15 error = tgo14

	index := func() (int, error) {
		a = b
		return step("I")
	}
	{
		var tgo12 int
		var tgo13 error
		tgo12, tgo13 = index()
		if tgo13 != nil {
			tgo15 = tgo13
			return tgo15
		}
		var tgo16 int
		var tgo17 error
		tgo16, tgo17 = step("V")
		if tgo17 != nil {
			tgo15 = tgo17
			return tgo15
		}
		a[tgo12] = tgo16
	}
	fmt.Print(a)
	return nil
}

type Box[T interface{ ~[2]int | ~[]int }] struct{ Value T }

func (b *Box[T]) Set() (tgo24 error) {
	type tgo18 = T
	var tgo25 error = tgo24

	defer func() {
		if recover() != nil {
			fmt.Print("r")
		}
	}()
	{
		var tgo19 *Box[T] = b
		var tgo20 bool = tgo2.TypeFor[tgo18]().Kind() == tgo2.Array
		var tgo21 tgo18
		if !tgo20 {
			tgo21 = tgo19.
				Value
		}
		var tgo22 int
		var tgo23 error
		tgo22, tgo23 = step("I")
		if tgo23 != nil {
			tgo25 = tgo23
			return tgo25
		}
		var tgo26 int
		var tgo27 error
		tgo26, tgo27 = step("V")
		if tgo27 != nil {
			tgo25 = tgo27
			return tgo25
		}
		var tgo28 int = tgo26
		(*func() *tgo18 {
			if tgo20 {
				return &tgo19.
					Value
			}
			return &tgo21
		}())[tgo22] = tgo28
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
