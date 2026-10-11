package main

import fmt_tgo6 "fmt"

import (
	"errors"
	"fmt"
)

var failure = errors.New("failed")

func read(value int) (int, error) {
	if value == 2 {
		return 0, failure
	}
	return value, nil
}

func propagate(wrapped bool) ([]int, error) {
	var tgo5 []int
	defer fmt.Println("deferred")
	if wrapped {
		{
			var tgo1 []int = []int{}
			{
				for x_tgo2 := range 3 {
					{
						var tgo3 int
						var tgo4 error
						tgo3, tgo4 = read(x_tgo2)
						if tgo4 != nil {
							return tgo5, fmt_tgo6.Errorf("read: %w", tgo4)
						}
						var tgo7 int = tgo3
						tgo1 = append(tgo1, tgo7)
					}
				}
			}
			return tgo1, nil
		}
	}
	{
		var tgo8 []int = []int{}
		{
			for x_tgo9 := range 3 {
				{
					var tgo10 int
					var tgo11 error
					tgo10, tgo11 = read(x_tgo9)
					if tgo11 != nil {
						return tgo5, tgo11
					}
					var tgo12 int = tgo10
					tgo8 = append(tgo8, tgo12)
				}
			}
		}
		return tgo8, nil
	}
}

func returnValue() int {
	var tgo13 []int = []int{}
	{
		for x_tgo14 := range 3 {
			{
				var tgo15 int
				if x_tgo14 == 1 {
					return 9
				} else {
					tgo15 = x_tgo14
				}
				var tgo16 int = tgo15
				tgo13 = append(tgo13, tgo16)
			}
		}
	}
	values := tgo13
	return len(values)
}

func main() {
	for _, wrapped := range []bool{false, true} {
		values, err := propagate(wrapped)
		fmt.Println(values, err, errors.Is(err, failure), err == failure)
	}
	fmt.Println(returnValue())
	{
		var tgo17 []int = []int{}
		{
			for x_tgo18 := range 3 {
				{
					var tgo19 int
					if x_tgo18 == 1 {
						continue
					} else {
						tgo19 = x_tgo18
					}
					var tgo20 int = tgo19
					tgo17 = append(tgo17, tgo20)
				}
			}
		}
		fmt.Println(tgo17)
	}
	{
		var tgo21 []int = []int{}
		{
			for x_tgo22 := range 3 {
				{
					var tgo24 bool
					if x_tgo22 == 1 {
						continue
					} else {
						tgo24 = true
					}
					if tgo24 {
						{
							var tgo23 int = x_tgo22
							tgo21 = append(tgo21, tgo23)
						}
					}
				}
			}
		}
		fmt.Println(tgo21)
	}
	{
		var tgo25 [][2]int = [][2]int{}
		{
			for x_tgo26 := range 2 {
				for y_tgo27 := range 3 {
					{
						var tgo29 int = x_tgo26
						var tgo28 int
						if y_tgo27 == 1 {
							break
						} else {
							tgo28 = y_tgo27
						}
						var tgo30 [2]int = [2]int{tgo29, tgo28}
						tgo25 = append(tgo25, tgo30)
					}
				}
			}
		}
		fmt.Println(tgo25)
	}
	for i := range 2 {
		var tgo31 []int = []int{}
		{
			{
				var tgo34 int
				if i == 0 {
					continue
				} else {
					tgo34 = 2
				}
				for x_tgo32 := range tgo34 {
					{
						var tgo33 int = x_tgo32
						tgo31 = append(tgo31, tgo33)
					}
				}
			}
		}
		values := tgo31
		fmt.Println("outer", i, values)
	}
outer:
	for i := range 2 {
		{
			var tgo35 []int = []int{}
			{
				for x_tgo36 := range 3 {
					{
						var tgo37 int
						if x_tgo36 == 1 {
							break outer
						} else {
							tgo37 = x_tgo36
						}
						var tgo38 int = tgo37
						tgo35 = append(tgo35, tgo38)
					}
				}
			}
			_ = tgo35
		}
		fmt.Println("unreached", i)
	}
	var result []int
	{
		var tgo39 []int = []int{}
		{
			for x_tgo40 := range 3 {
				{
					var tgo41 int
					if x_tgo40 == 1 {
						goto done
					} else {
						tgo41 = x_tgo40
					}
					var tgo42 int = tgo41
					tgo39 = append(tgo39, tgo42)
				}
			}
		}
		result = tgo39
	}
	fmt.Println("unreached")
done:
	fmt.Println(result)
}
