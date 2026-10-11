package main

import "fmt"

var trace []string

func mark(label string, value int) int {
	trace = append(trace, label)
	return value
}

func mapPanic() {
	defer func() { fmt.Println(recover() != nil, trace); trace = nil }()
	{
		var tgo1 map[any]int = map[any]int{}
		{
			for x_tgo2 := range 1 {
				{
					var tgo3 any = []int{x_tgo2}
					var tgo4 int = mark("value before hash", x_tgo2)
					tgo1[tgo3] = tgo4
				}
			}
		}
		_ = tgo1
	}
}

func deferred() {
	{
		var tgo5 []int = []int{}
		{
			for x_tgo6 := range 3 {
				{
					var tgo7 int
					if true {
						defer fmt.Println("visit", x_tgo6)
						tgo7 = x_tgo6
					} else {
						tgo7 = 0
					}
					var tgo8 int = tgo7
					tgo5 = append(tgo5, tgo8)
				}
			}
		}
		fmt.Println(tgo5)
	}
}

func main() {
	mapPanic()
	deferred()
	source := [3]int{1, 2, 3}
	mapping := map[string]int{"a": 1}
	var tgo9 map[string]int = map[string]int{}
	{
		for key_tgo10, value_tgo11 := range mapping {
			{
				var tgo13 string = key_tgo10
				var tgo12 int
				if true {
					delete(mapping, key_tgo10)
					tgo12 = value_tgo11
				} else {
					tgo12 = 0
				}
				var tgo14 string = tgo13
				var tgo15 int = tgo12
				tgo9[tgo14] = tgo15
			}
		}
	}
	retained := tgo9
	fmt.Println(retained, mapping)
	var tgo18 [3]int = (source)
	var tgo16 []int = make([]int, len(tgo18))
	{
		for tgo19, x_tgo17 := range tgo18 {
			{
				var tgo20 int
				if x_tgo17 == 1 {
					source[1] = 9
					tgo20 = x_tgo17
				} else {
					tgo20 = x_tgo17
				}
				var tgo21 int = tgo20
				tgo16[tgo19] = tgo21
			}
		}
	}
	copied := tgo16
	fmt.Println(copied, source)
	outer := []int{1, 2}
	{
		var tgo24 []int = (outer)
		var tgo22 []int = make([]int, len(tgo24))
		{
			for tgo25, outer_tgo23 := range tgo24 {
				{
					var tgo26 int = outer_tgo23
					tgo22[tgo25] = tgo26
				}
			}
		}
		fmt.Println(tgo22)
	}
	stores := [1][]int{}
	{
		var tgo27 int = mark("destination", 0)
		var tgo28 []int = []int{}
		{
			for x_tgo29 := range 2 {
				{
					var tgo30 int = mark("output", x_tgo29)
					tgo28 = append(tgo28, tgo30)
				}
			}
		}
		stores[tgo27] = tgo28
	}
	fmt.Println(stores, trace)
	trace = nil
	channel := make(chan string, 1)
	{
		var tgo34 func(values []int) = func(values []int) { channel <- fmt.Sprint(values) }
		var tgo31 []int = []int{}
		{
			for x_tgo32 := range 2 {
				{
					var tgo33 int = mark("go argument", x_tgo32)
					tgo31 = append(tgo31, tgo33)
				}
			}
		}
		go tgo34(tgo31)
	}
	fmt.Println(<-channel, trace)
	trace = nil
	ready := make(chan int, 1)
	ready <- 7
	{
		var tgo35 []chan int = []chan int{}

		{
			for range 1 {
				{
					var tgo36 chan int
					if true {
						trace = append(trace, "select operand")
						tgo36 = ready
					} else {
						tgo36 = nil
					}
					var tgo37 chan int = tgo36
					tgo35 = append(tgo35, tgo37)
				}
			}
		}
		var tgo38 chan int = (tgo35)[0]
		var tgo39 int
		select {
		case tgo39 = <-tgo38:
			value := tgo39
			fmt.Println(value, trace)
		}
	}

}
