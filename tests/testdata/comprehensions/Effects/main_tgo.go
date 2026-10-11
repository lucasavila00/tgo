package main

import "fmt"

var trace []string

func mark(label string, value int) int {
	trace = append(trace, label)
	return value
}

func source(label string, values []int) []int {
	trace = append(trace, label)
	return values
}

func show(first int, values []int, last int) {
	fmt.Println(first, values, last, trace)
	trace = nil
}

func main() {
	{
		var tgo4 int = mark("left", 1)
		var tgo1 []int = []int{}
		{
			for _, x_tgo2 := range source("source", []int{1, 2}) {
				if mark("filter", x_tgo2) > 1 {
					{
						var tgo3 int = mark("value", x_tgo2)
						tgo1 = append(tgo1, tgo3)
					}
				}
			}
		}
		show(tgo4, tgo1, mark("right", 3))
	}
	var tgo5 [][2]int = [][2]int{}
	{
		for _, x_tgo6 := range source("outer", []int{0, 2}) {
			if mark("outer filter", x_tgo6) > 0 {
				for _, y_tgo7 := range source("inner", []int{x_tgo6, x_tgo6 + 1}) {
					{
						var tgo8 [2]int = [2]int{x_tgo6, mark("inner value", y_tgo7)}
						tgo5 = append(tgo5, tgo8)
					}
				}
			}
		}
	}
	result := tgo5
	fmt.Println(result, trace)
	trace = nil
	{
		var tgo13 []int = (source("nested", []int{1, 2}))
		var tgo12 []int = make([]int, len(tgo13))
		{
			for tgo14, y_tgo11 := range tgo13 {
				{
					var tgo15 int = mark("produce", y_tgo11)
					tgo12[tgo14] = tgo15
				}
			}
		}
		var tgo16 []int = (tgo12)
		var tgo9 []int = make([]int, len(tgo16))
		{
			for tgo17, x_tgo10 := range tgo16 {
				{
					var tgo18 int = mark("consume", x_tgo10)
					tgo9[tgo17] = tgo18
				}
			}
		}
		fmt.Println(tgo9, trace)
	}
	trace = nil
	{
		var tgo19 map[int]int = map[int]int{}
		{
			for _, x_tgo20 := range []int{1, 2} {
				{
					var tgo21 int = mark("key", 0)
					var tgo22 int = mark("value", x_tgo20)
					tgo19[tgo21] = tgo22
				}
			}
		}
		fmt.Println(tgo19, trace)
	}
	trace = nil
	xs := []int{1, 2, 3}
	var tgo25 [3]int = ([3]int{1, 2, 3})
	var tgo23 []int = make([]int, len(tgo25))
	{
		for tgo26, x_tgo24 := range tgo25 {
			{
				var tgo27 int
				if x_tgo24 == 1 {
					xs[1] = 9
					tgo27 = x_tgo24
				} else {
					tgo27 = x_tgo24
				}
				var tgo28 int = tgo27
				tgo23[tgo26] = tgo28
			}
		}
	}
	copied := tgo23
	fmt.Println(copied, xs)
	{
		var tgo31 []int = ([]int{7, 8})
		var tgo29 []int = make([]int, len(tgo31))
		{
			for tgo32, x_tgo30 := range tgo31 {
				{
					var tgo33 int = mark("defer", x_tgo30)
					tgo29[tgo32] = tgo33
				}
			}
		}
		defer fmt.Println(tgo29, trace)
	}
	{
		var tgo37 bool
		tgo37 = false
		if tgo37 {
			var tgo34 []int = []int{}
			{
				for x_tgo35 := range 1 {
					{
						var tgo36 int = mark("unreached", x_tgo35)
						tgo34 = append(tgo34, tgo36)
					}
				}
			}
			tgo37 = len(tgo34) > 0
		}
		fmt.Println(tgo37 == (0 == 0), trace)
	}
}
