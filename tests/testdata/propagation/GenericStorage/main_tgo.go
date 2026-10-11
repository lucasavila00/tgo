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
	var tgo9 error = tgo8

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
			tgo8 = tgo6
			if tgo8 != tgo9 {
				return tgo8
			}
			var tgo10 int
			var tgo11 error
			tgo10, tgo11 = step("V")
			tgo8 = tgo11
			if tgo8 != tgo9 {
				return tgo8
			}
			var tgo12 int = tgo10
			(*func() *tgo1 {
				if tgo3 {
					return &a
				}
				return &tgo4
			}())[tgo5] = tgo12
		}
	}
	fmt.Print(a)
	return nil

}

func arrays[T interface{ ~[2]int | ~[3]int }](a, b T) (tgo15 error) {
	var tgo16 error = tgo15
	var tgo17 error = tgo16

	index := func() (int, error) {
		a = b
		return step("I")
	}
	{
		var tgo13 int
		var tgo14 error
		tgo13, tgo14 = index()
		tgo16 = tgo14
		if tgo16 != tgo17 {
			return tgo16
		}
		var tgo18 int
		var tgo19 error
		tgo18, tgo19 = step("V")
		tgo16 = tgo19
		if tgo16 != tgo17 {
			return tgo16
		}
		a[tgo13] = tgo18
	}
	fmt.Print(a)
	return nil
}

type Box[T interface{ ~[2]int | ~[]int }] struct{ Value T }

func (b *Box[T]) Set() (tgo26 error) {
	type tgo20 = T
	var tgo27 error = tgo26
	var tgo28 error = tgo27

	defer func() {
		if recover() != nil {
			fmt.Print("r")
		}
	}()
	{
		var tgo21 *Box[T] = b
		var tgo22 bool = tgo2.TypeFor[tgo20]().Kind() == tgo2.Array
		var tgo23 tgo20
		if !tgo22 {
			tgo23 = tgo21.
				Value
		}
		var tgo24 int
		var tgo25 error
		tgo24, tgo25 = step("I")
		tgo27 = tgo25
		if tgo27 != tgo28 {
			return tgo27
		}
		var tgo29 int
		var tgo30 error
		tgo29, tgo30 = step("V")
		tgo27 = tgo30
		if tgo27 != tgo28 {
			return tgo27
		}
		var tgo31 int = tgo29
		(*func() *tgo20 {
			if tgo22 {
				return &tgo21.
					Value
			}
			return &tgo23
		}())[tgo24] = tgo31
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
