package main

import (
	"fmt"
	"unsafe"
)

var effects int

func main() {
	{
		var tgo1 []int = []int{}
		{
			for i_tgo2 := range 2 {
				{
					var tgo3 int
					if true {
						effects++
						tgo3 = i_tgo2
					} else {
						tgo3 = 0
					}
					var tgo4 int = tgo3
					tgo1 = append(tgo1, tgo4)
				}
			}
		}
		fmt.Println(len([1][]int{tgo1}), effects)
	}
	{
		var tgo5 []int = []int{}
		{
			for i_tgo6 := range 2 {
				{
					var tgo7 int
					if true {
						effects++
						tgo7 = i_tgo6
					} else {
						tgo7 = 0
					}
					var tgo8 int = tgo7
					tgo5 = append(tgo5, tgo8)
				}
			}
		}
		fmt.Println(cap([1][]int{tgo5}), effects)
	}
	{
		var tgo9 []int = []int{}
		{
			for i_tgo10 := range 2 {
				{
					var tgo11 int
					if true {
						effects++
						tgo11 = i_tgo10
					} else {
						tgo11 = 0
					}
					var tgo12 int = tgo11
					tgo9 = append(tgo9, tgo12)
				}
			}
		}
		for range [1][]int{tgo9} {
			fmt.Println("range", effects)
		}
	}
	{
		if 0 != 0 {
			var tgo13 []int = []int{}
			{
				for i_tgo14 := range 2 {
					{
						var tgo15 int
						if true {
							effects++
							tgo15 = i_tgo14
						} else {
							tgo15 = 0
						}
						var tgo16 int = tgo15
						tgo13 = append(tgo13, tgo16)
					}
				}
			}
			_ = tgo13
		}
		fmt.Println(unsafe.Sizeof(*func() (tgo17 *[]int) {
			return
		}()), effects)
	}
	{
		if 0 != 0 {
			var tgo18 map[int]int = map[int]int{}
			{
				for i_tgo19 := range 2 {
					{
						var tgo21 int = i_tgo19
						var tgo20 int
						if true {
							effects++
							tgo20 = i_tgo19
						} else {
							tgo20 = 0
						}
						var tgo22 int = tgo21
						var tgo23 int = tgo20
						tgo18[tgo22] = tgo23
					}
				}
			}
			_ = tgo18
		}
		fmt.Println(unsafe.Alignof(*func() (tgo24 *map[int]int) {
			return
		}()), effects)
	}
	{
		if 0 != 0 {
			var tgo25 []int = []int{}

			{
				for i_tgo26 := range 2 {
					{
						var tgo27 int
						if true {
							effects++
							tgo27 = i_tgo26
						} else {
							tgo27 = 0
						}
						var tgo28 int = tgo27
						tgo25 = append(tgo25, tgo28)
					}
				}
			}
			_ = (struct{ Values []int }{tgo25}).Values
		}
		fmt.Println(unsafe.Offsetof((*func() (tgo29 *struct {
			Values []int
		}) {
			return
		}()).
			Values), effects)
	}
	{
		var tgo30 []int = make([]int, 2)
		{
			for i_tgo31 := range [2]int{1, 2} {
				{
					var tgo33 int = i_tgo31
					tgo30[i_tgo31] = tgo33
				}
			}
		}
		fmt.Println(tgo30)
	}
	var pointer *[2]int
	{
		var tgo34 []int = make([]int, 2)
		{
			for i_tgo35 := range *pointer {
				{
					var tgo37 int = i_tgo35
					tgo34[i_tgo35] = tgo37
				}
			}
		}
		fmt.Println(tgo34)
	}
}
