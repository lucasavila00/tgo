package main

import slices_tgo3 "slices"

import "fmt"

type Number int
type Numbers []Number
type Lookup map[int]Number

var initial = func() Numbers {
	var tgo1 Numbers = Numbers{}
	var tgo4 int = 0
	{
		for x_tgo2 := range 3 {
			{
				var tgo5 Number = Number(x_tgo2)
				tgo1 = slices_tgo3.Grow(tgo1, 1)
				tgo1 = tgo1[:tgo4+1]
				tgo1[tgo4] = tgo5
				tgo4++
			}
		}
	}
	return tgo1
}()

func twice[S ~[]E, E ~int](values S) S {
	{
		var tgo6 S = S{}
		var tgo8 int = 0
		{
			for _, x_tgo7 := range values {
				{
					var tgo9 E = x_tgo7 * 2
					tgo6 = slices_tgo3.Grow(tgo6, 1)
					tgo6 = tgo6[:tgo8+1]
					tgo6[tgo8] = tgo9
					tgo8++
				}
			}
		}
		return tgo6
	}
}

func copyMap[M ~map[K]V, K comparable, V any](values M) M {
	{
		var tgo10 M = M{}
		{
			for k_tgo11, v_tgo12 := range values {
				{
					var tgo13 K = k_tgo11
					var tgo14 V = v_tgo12
					tgo10[tgo13] = tgo14
				}
			}
		}
		return tgo10
	}
}

func main() {
	xs := Numbers{2, 3}
	fmt.Println(initial, twice(xs), copyMap(Lookup{1: 2}))
	{
		var tgo15 []uint64 = []uint64{}
		var tgo17 int = 0
		{
			for n_tgo16 := range 2 {
				{
					var tgo18 uint64 = 1 << n_tgo16
					tgo15 = slices_tgo3.Grow(tgo15, 1)
					tgo15 = tgo15[:tgo17+1]
					tgo15[tgo17] = tgo18
					tgo17++
				}
			}
		}
		fmt.Println(tgo15)
	}
	{
		var tgo19 []*int = []*int{}
		var tgo20 int = 0
		{
			for range 2 {
				{
					var tgo21 *int = nil
					tgo19 = slices_tgo3.Grow(tgo19, 1)
					tgo19 = tgo19[:tgo20+1]
					tgo19[tgo20] = tgo21
					tgo20++
				}
			}
		}
		fmt.Println(tgo19)
	}
	{
		var tgo22 []any = []any{}
		var tgo24 int = 0
		{
			for _, x_tgo23 := range xs {
				{
					var tgo25 any
					if x_tgo23 > 2 {
						tgo25 = x_tgo23
					} else {
						tgo25 = "small"
					}
					var tgo26 any = tgo25
					tgo22 = slices_tgo3.Grow(tgo22, 1)
					tgo22 = tgo22[:tgo24+1]
					tgo22[tgo24] = tgo26
					tgo24++
				}
			}
		}
		fmt.Println(tgo22)
	}
	{
		var tgo27 map[any]any = map[any]any{}
		{
			for _, x_tgo28 := range xs {
				{
					var tgo29 any = x_tgo28
					var tgo30 any = x_tgo28
					tgo27[tgo29] = tgo30
				}
			}
		}
		fmt.Println(tgo27)
	}
	var tgo31 Numbers = Numbers{}
	var tgo33 int = 0
	{
		for _, x_tgo32 := range xs {
			{
				var tgo34 Number = x_tgo32
				tgo31 = slices_tgo3.Grow(tgo31, 1)
				tgo31 = tgo31[:tgo33+1]
				tgo31[tgo33] = tgo34
				tgo33++
			}
		}
	}
	var tgo35 *Numbers = &tgo31
	pointer := tgo35
	fmt.Println(*pointer)
	{
		var tgo36 Numbers = Numbers{}
		var tgo38 int = 0
		{
			for _, Numbers_tgo37 := range xs {
				{
					var tgo39 Number = Numbers_tgo37
					tgo36 = slices_tgo3.Grow(tgo36, 1)
					tgo36 = tgo36[:tgo38+1]
					tgo36[tgo38] = tgo39
					tgo38++
				}
			}
		}
		fmt.Println(tgo36)
	}
	append := "outer"
	{
		var tgo40 []Number = []Number{}
		var tgo42 int = 0
		{
			for _, x_tgo41 := range xs {
				if x_tgo41 > 0 {
					{
						var tgo43 Number = x_tgo41
						tgo40 = slices_tgo3.Grow(tgo40, 1)
						tgo40 = tgo40[:tgo42+1]
						tgo40[tgo42] = tgo43
						tgo42++
					}
				}
			}
		}
		fmt.Println(tgo40, append)
	}
	{
		var tgo44 []int = []int{}
		var tgo46 int = 0
		{
			for append_tgo45 := range 2 {
				{
					var tgo47 int = append_tgo45
					tgo44 = slices_tgo3.Grow(tgo44, 1)
					tgo44 = tgo44[:tgo46+1]
					tgo44[tgo46] = tgo47
					tgo46++
				}
			}
		}
		fmt.Println(tgo44)
	}
	{
		var tgo48 []int = []int{}
		var tgo50 int = 0
		{
			for len_tgo49 := range 2 {
				{
					var tgo51 int = len_tgo49
					tgo48 = slices_tgo3.Grow(tgo48, 1)
					tgo48 = tgo48[:tgo50+1]
					tgo48[tgo50] = tgo51
					tgo50++
				}
			}
		}
		fmt.Println(tgo48)
	}
	{
		var tgo52 []int = []int{}
		var tgo54 int = 0
		{
			for make_tgo53 := range 2 {
				{
					var tgo55 int = make_tgo53
					tgo52 = slices_tgo3.Grow(tgo52, 1)
					tgo52 = tgo52[:tgo54+1]
					tgo52[tgo54] = tgo55
					tgo54++
				}
			}
		}
		fmt.Println(tgo52)
	}
	var absent Lookup
	var tgo56 Lookup = Lookup{}
	{
		for k_tgo57, v_tgo58 := range absent {
			{
				var tgo59 int = k_tgo57
				var tgo60 Number = v_tgo58
				tgo56[tgo59] = tgo60
			}
		}
	}
	empty := tgo56
	fmt.Println(empty == nil)
}
