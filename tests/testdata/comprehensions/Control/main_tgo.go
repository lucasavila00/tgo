package main

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
	var zero_tgo5 []int
	defer fmt.Println("deferred")
	if wrapped {
		var tgo1 []int = []int{}
		{
			for x_tgo2 := range 3 {
				var value_tgo3 int
				var error_tgo4 error
				value_tgo3, error_tgo4 = read(x_tgo2)
				if error_tgo4 != nil {
					return zero_tgo5, fmt.Errorf("read: %w", error_tgo4)
				}
				var tgo6 int = value_tgo3
				tgo1 = append(tgo1, tgo6)
			}
		}
		return tgo1, nil
	}
	var tgo7 []int = []int{}
	{
		for x_tgo8 := range 3 {
			var value_tgo9 int
			var error_tgo10 error
			value_tgo9, error_tgo10 = read(x_tgo8)
			if error_tgo10 != nil {
				return zero_tgo5, error_tgo10
			}
			var tgo11 int = value_tgo9
			tgo7 = append(tgo7, tgo11)
		}
	}
	return tgo7, nil
}

func returnValue() int {
	var tgo12 []int = []int{}
	{
		for x_tgo13 := range 3 {
			var tgo14 int
			if x_tgo13 == 1 {
				return 9
			} else {
				tgo14 = x_tgo13
			}
			var tgo15 int = tgo14
			tgo12 = append(tgo12, tgo15)
		}
	}
	values := tgo12
	return len(values)
}

func main() {
	for _, wrapped := range []bool{false, true} {
		values, err := propagate(wrapped)
		fmt.Println(values, err, errors.Is(err, failure), err == failure)
	}
	fmt.Println(returnValue())
	{
		var tgo16 []int = []int{}
		{
			for x_tgo17 := range 3 {
				{
					var tgo18 int
					if x_tgo17 == 1 {
						continue
					} else {
						tgo18 = x_tgo17
					}
					var tgo19 int = tgo18
					tgo16 = append(tgo16, tgo19)
				}
			}
		}
		fmt.Println(tgo16)
	}
	{
		var tgo20 []int = []int{}
		{
			for x_tgo21 := range 3 {
				{
					var tgo23 bool
					if x_tgo21 == 1 {
						continue
					} else {
						tgo23 = true
					}
					if tgo23 {
						{
							var tgo22 int = x_tgo21
							tgo20 = append(tgo20, tgo22)
						}
					}
				}
			}
		}
		fmt.Println(tgo20)
	}
	{
		var tgo24 [][2]int = [][2]int{}
		{
			for x_tgo25 := range 2 {
				for y_tgo26 := range 3 {
					{
						var value_tgo28 int = x_tgo25
						var tgo27 int
						if y_tgo26 == 1 {
							break
						} else {
							tgo27 = y_tgo26
						}
						var tgo29 [2]int = [2]int{value_tgo28, tgo27}
						tgo24 = append(tgo24, tgo29)
					}
				}
			}
		}
		fmt.Println(tgo24)
	}
	for i := range 2 {
		var tgo30 []int = []int{}
		{
			{
				var tgo33 int
				if i == 0 {
					continue
				} else {
					tgo33 = 2
				}
				for x_tgo31 := range tgo33 {
					{
						var tgo32 int = x_tgo31
						tgo30 = append(tgo30, tgo32)
					}
				}
			}
		}
		values := tgo30
		fmt.Println("outer", i, values)
	}
outer:
	for i := range 2 {
		{
			var tgo34 []int = []int{}
			{
				for x_tgo35 := range 3 {
					{
						var tgo36 int
						if x_tgo35 == 1 {
							break outer
						} else {
							tgo36 = x_tgo35
						}
						var tgo37 int = tgo36
						tgo34 = append(tgo34, tgo37)
					}
				}
			}
			_ = tgo34
		}
		fmt.Println("unreached", i)
	}
	var result []int
	{
		var tgo38 []int = []int{}
		{
			for x_tgo39 := range 3 {
				{
					var tgo40 int
					if x_tgo39 == 1 {
						goto done
					} else {
						tgo40 = x_tgo39
					}
					var tgo41 int = tgo40
					tgo38 = append(tgo38, tgo41)
				}
			}
		}
		result = tgo38
	}
	fmt.Println("unreached")
done:
	fmt.Println(result)
}
