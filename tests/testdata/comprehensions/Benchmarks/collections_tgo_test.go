package main

import "testing"

var sliceResult []int
var mapResult map[int]int

func denseComprehension(values []int) []int {
	{
		var tgo3 []int = (values)
		var tgo1 []int = make([]int, len(tgo3))
		{
			for tgo4, x_tgo2 := range tgo3 {
				{
					var tgo5 int = x_tgo2 * 2
					tgo1[tgo4] = tgo5
				}
			}
		}
		return tgo1
	}
}

func denseGo(values []int) []int {
	result := make([]int, len(values))
	for i, x := range values {
		result[i] = x * 2
	}
	return result
}

func sparseComprehension(values []int) []int {
	{
		var tgo6 []int = []int{}
		{
			for _, x_tgo7 := range values {
				if x_tgo7%100 == 0 {
					{
						var tgo8 int = x_tgo7 * 2
						tgo6 = append(tgo6, tgo8)
					}
				}
			}
		}
		return tgo6
	}
}

func sparseGo(values []int) []int {
	result := []int{}
	for _, x := range values {
		if x%100 == 0 {
			result = append(result, x*2)
		}
	}
	return result
}

func materialized(values []int) []int {
	filtered := []int{}
	for _, x := range values {
		if x%100 == 0 {
			filtered = append(filtered, x)
		}
	}
	result := make([]int, len(filtered))
	for i, x := range filtered {
		result[i] = x * 2
	}
	return result
}

func nestedComprehension(values []int) []int {
	{
		var tgo9 []int = []int{}
		{
			for _, x_tgo10 := range values {
				for y_tgo11 := range 4 {
					{
						var tgo12 int = x_tgo10 + y_tgo11
						tgo9 = append(tgo9, tgo12)
					}
				}
			}
		}
		return tgo9
	}
}

func nestedGo(values []int) []int {
	result := []int{}
	for _, x := range values {
		for y := range 4 {
			result = append(result, x+y)
		}
	}
	return result
}

func repeatedComprehension(values []int) map[int]int {
	{
		var tgo13 map[int]int = map[int]int{}
		{
			for _, x_tgo14 := range values {
				{
					var tgo15 int = x_tgo14 % 10
					var tgo16 int = x_tgo14
					tgo13[tgo15] = tgo16
				}
			}
		}
		return tgo13
	}
}

func repeatedGo(values []int) map[int]int {
	result := map[int]int{}
	for _, x := range values {
		result[x%10] = x
	}
	return result
}

func iterator(values []int) func(func(int) bool) {
	return func(yield func(int) bool) {
		for _, value := range values {
			if !yield(value) {
				return
			}
		}
	}
}

func iteratorComprehension(values []int) []int {
	{
		var tgo17 []int = []int{}
		{
			for x_tgo18 := range iterator(values) {
				{
					var tgo19 int = x_tgo18 * 2
					tgo17 = append(tgo17, tgo19)
				}
			}
		}
		return tgo17
	}
}

func iteratorGo(values []int) []int {
	result := []int{}
	for x := range iterator(values) {
		result = append(result, x*2)
	}
	return result
}

func BenchmarkCollections(b *testing.B) {
	values := make([]int, 10000)
	for i := range values {
		values[i] = i
	}
	for _, test := range []struct {
		name string
		run  func([]int) []int
	}{
		{"dense/comprehension", denseComprehension},
		{"dense/Go", denseGo},
		{"sparse/comprehension", sparseComprehension},
		{"sparse/Go", sparseGo},
		{"sparse/materialized", materialized},
		{"nested/comprehension", nestedComprehension},
		{"nested/Go", nestedGo},
		{"iterator/comprehension", iteratorComprehension},
		{"iterator/Go", iteratorGo},
	} {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sliceResult = test.run(values)
			}
		})
	}
	for _, test := range []struct {
		name string
		run  func([]int) map[int]int
	}{
		{"repeated/comprehension", repeatedComprehension},
		{"repeated/Go", repeatedGo},
	} {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				mapResult = test.run(values)
			}
		})
	}
}
