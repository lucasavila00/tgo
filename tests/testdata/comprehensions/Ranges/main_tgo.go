package main

import "fmt"

func sequence(yield func(int) bool) {
	for _, value := range []int{2, 4, 6} {
		if !yield(value) {
			fmt.Println("iterator stopped")
			return
		}
	}
}

func pairs(yield func(int, string) bool) {
	for i, value := range []string{"a", "b"} {
		if !yield(i, value) {
			return
		}
	}
}

func visits(yield func() bool) {
	for range 2 {
		if !yield() {
			return
		}
	}
}

func main() {
	xs := []int{1, 2, 3}
	{
		var tgo3 []int = (xs)
		var tgo1 []int = make([]int, len(tgo3))
		{
			for tgo4, x_tgo2 := range tgo3 {
				{
					var tgo5 int = x_tgo2 * 2
					tgo1[tgo4] = tgo5
				}
			}
		}
		fmt.Println(tgo1)
	}
	{
		var tgo8 [3]int = ([3]int{4, 5, 6})
		var tgo6 []int = make([]int, len(tgo8))
		{
			for tgo9, x_tgo7 := range tgo8 {
				{
					var tgo10 int = x_tgo7
					tgo6[tgo9] = tgo10
				}
			}
		}
		fmt.Println(tgo6)
	}
	{
		var tgo11 []int = []int{}
		{
			for i_tgo12 := range (*[3]int)(nil) {
				{
					var tgo13 int = i_tgo12
					tgo11 = append(tgo11, tgo13)
				}
			}
		}
		fmt.Println(tgo11)
	}
	{
		var tgo14 []rune = []rune{}
		{
			for _, r_tgo15 := range "aé" {
				{
					var tgo16 rune = r_tgo15
					tgo14 = append(tgo14, tgo16)
				}
			}
		}
		fmt.Println(tgo14)
	}
	{
		var tgo17 map[int]string = map[int]string{}
		{
			for i_tgo18, value_tgo19 := range pairs {
				{
					var tgo20 int = i_tgo18
					var tgo21 string = value_tgo19
					tgo17[tgo20] = tgo21
				}
			}
		}
		fmt.Println(tgo17)
	}
	{
		var tgo22 []int = []int{}
		{
			for x_tgo23 := range sequence {
				if x_tgo23 < 5 {
					{
						var tgo24 int = x_tgo23
						tgo22 = append(tgo22, tgo24)
					}
				}
			}
		}
		fmt.Println(tgo22)
	}
	{
		var tgo25 []int = []int{}
		{
			for x_tgo26 := range sequence {
				{
					var tgo27 int
					if x_tgo26 > 2 {
						break
					} else {
						tgo27 = x_tgo26
					}
					var tgo28 int = tgo27
					tgo25 = append(tgo25, tgo28)
				}
			}
		}
		fmt.Println(tgo25)
	}
	{
		var tgo29 []int = []int{}
		{
			for i_tgo30 := range 3 {
				{
					var tgo31 int = i_tgo30
					tgo29 = append(tgo29, tgo31)
				}
			}
		}
		fmt.Println(tgo29)
	}
	{
		var tgo32 []int = []int{}
		{
			for range visits {
				{
					var tgo33 int = 1
					tgo32 = append(tgo32, tgo33)
				}
			}
		}
		fmt.Println(tgo32)
	}
	channel := make(chan int, 2)
	channel <- 8
	channel <- 9
	close(channel)
	{
		var tgo34 []int = []int{}
		{
			for x_tgo35 := range channel {
				{
					var tgo36 int = x_tgo35
					tgo34 = append(tgo34, tgo36)
				}
			}
		}
		fmt.Println(tgo34)
	}
	{
		var tgo37 map[string]int = map[string]int{}
		{
			for key_tgo38, value_tgo39 := range map[string]int{"a": 1, "b": 2} {
				{
					var tgo40 string = key_tgo38
					var tgo41 int = value_tgo39 * 2
					tgo37[tgo40] = tgo41
				}
			}
		}
		fmt.Println(tgo37)
	}
	{
		var tgo45 []int = (xs)
		var tgo42 map[int]int = make(map[int]int, len(tgo45))
		{
			for i_tgo43, x_tgo44 := range tgo45 {
				{
					var tgo46 int = i_tgo43
					var tgo47 int = x_tgo44 * 3
					tgo42[tgo46] = tgo47
				}
			}
		}
		fmt.Println(tgo42)
	}
	{
		var tgo48 [][2]int = [][2]int{}
		{
			for _, x_tgo49 := range xs {
				if x_tgo49 > 1 {
					for y_tgo50 := range x_tgo49 {
						{
							var tgo51 [2]int = [2]int{x_tgo49, y_tgo50}
							tgo48 = append(tgo48, tgo51)
						}
					}
				}
			}
		}
		fmt.Println(tgo48)
	}
	var absent []int
	var tgo54 []int = (absent)
	var tgo52 []int = make([]int, len(tgo54))
	{
		for tgo55, x_tgo53 := range tgo54 {
			{
				var tgo56 int = x_tgo53
				tgo52[tgo55] = tgo56
			}
		}
	}
	empty := tgo52
	fmt.Println(empty == nil, len(empty))
	var tgo59 []int = (xs)
	var tgo57 []func() int = make([]func() int, len(tgo59))
	{
		for tgo60, x_tgo58 := range tgo59 {
			{
				var tgo61 func() int = func() int { return x_tgo58 }
				tgo57[tgo60] = tgo61
			}
		}
	}
	closures := tgo57
	fmt.Println(closures[0](), closures[1](), closures[2]())
}
