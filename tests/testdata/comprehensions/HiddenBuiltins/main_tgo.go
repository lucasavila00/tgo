package main

import slices_tgo3 "slices"

import "fmt"

func main() {
	xs := []int{1, 2, 3}
	var tgo1 []int = []int{}
	var tgo4 int = 0
	{
		for _, x_tgo2 := range xs {
			var tgo5 int = x_tgo2 * 2
			tgo1 = slices_tgo3.Grow(tgo1, 1)
			tgo1 = tgo1[:tgo4+1]
			tgo1[tgo4] = tgo5
			tgo4++
		}
	}
	fmt.Println(tgo1, append, make, len)
	var tgo6 map[int]int = map[int]int{}
	{
		for i_tgo7, x_tgo8 := range xs {
			var tgo9 int = i_tgo7
			var tgo10 int = x_tgo8
			tgo6[tgo9] = tgo10
		}
	}
	fmt.Println(tgo6)
}
