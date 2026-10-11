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
	var tgo25 []int = ([]int{4, 5, 6})
	var tgo22 []int = make([]int, len(tgo25))
	{
		for i_tgo23, x_tgo24 := range tgo25 {
			var tgo26 int = i_tgo23
			{
				var tgo27 int
				if true {
					i_tgo23 = 0
					tgo27 = x_tgo24 + i_tgo23
				} else {
					tgo27 = x_tgo24
				}
				var tgo28 int = tgo27
				tgo22[tgo26] = tgo28
			}
		}
	}
	indices := tgo22
	fmt.Println(indices)
	{
		var tgo29 []int = make([]int, 2)
		{
			for i_tgo30 := range [2]int{} {
				var tgo31 int = i_tgo30
				{
					var tgo32 int
					if true {
						i_tgo30 = 10
						tgo32 = i_tgo30
					} else {
						tgo32 = 0
					}
					var tgo33 int = tgo32
					tgo29[tgo31] = tgo33
				}
			}
		}
		fmt.Println(tgo29)
	}
	outer := []int{1, 2}
	{
		var tgo36 []int = (outer)
		var tgo34 []int = make([]int, len(tgo36))
		{
			for tgo37, outer_tgo35 := range tgo36 {
				{
					var tgo38 int = outer_tgo35
					tgo34[tgo37] = tgo38
				}
			}
		}
		fmt.Println(tgo34)
	}
	stores := [1][]int{}
	{
		var tgo39 int = mark("destination", 0)
		var tgo40 []int = []int{}
		{
			for x_tgo41 := range 2 {
				{
					var tgo42 int = mark("output", x_tgo41)
					tgo40 = append(tgo40, tgo42)
				}
			}
		}
		stores[tgo39] = tgo40
	}
	fmt.Println(stores, trace)
	trace = nil
	channel := make(chan string, 1)
	{
		var tgo46 func(values []int) = func(values []int) { channel <- fmt.Sprint(values) }
		var tgo43 []int = []int{}
		{
			for x_tgo44 := range 2 {
				{
					var tgo45 int = mark("go argument", x_tgo44)
					tgo43 = append(tgo43, tgo45)
				}
			}
		}
		go tgo46(tgo43)
	}
	fmt.Println(<-channel, trace)
	trace = nil
	ready := make(chan int, 1)
	ready <- 7
	{
		var tgo47 []chan int = []chan int{}

		{
			for range 1 {
				{
					var tgo48 chan int
					if true {
						trace = append(trace, "select operand")
						tgo48 = ready
					} else {
						tgo48 = nil
					}
					var tgo49 chan int = tgo48
					tgo47 = append(tgo47, tgo49)
				}
			}
		}
		var tgo50 chan int = (tgo47)[0]
		var tgo51 int
		select {
		case tgo51 = <-tgo50:
			value := tgo51
			fmt.Println(value, trace)
		}
	}

}
