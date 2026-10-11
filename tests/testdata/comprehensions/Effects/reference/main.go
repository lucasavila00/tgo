package main

import "fmt"

var trace []string

func mark(label string, value int) int        { trace = append(trace, label); return value }
func source(label string, values []int) []int { trace = append(trace, label); return values }
func show(first int, values []int, last int)  { fmt.Println(first, values, last, trace); trace = nil }
func main() {
	first := mark("left", 1)
	values := []int{}
	for _, x := range source("source", []int{1, 2}) {
		if mark("filter", x) > 1 {
			values = append(values, mark("value", x))
		}
	}
	show(first, values, mark("right", 3))
	result := [][2]int{}
	for _, x := range source("outer", []int{0, 2}) {
		if mark("outer filter", x) > 0 {
			for _, y := range source("inner", []int{x, x + 1}) {
				result = append(result, [2]int{x, mark("inner value", y)})
			}
		}
	}
	fmt.Println(result, trace)
	trace = nil
	produced := []int{}
	for _, y := range source("nested", []int{1, 2}) {
		produced = append(produced, mark("produce", y))
	}
	consumed := []int{}
	for _, x := range produced {
		consumed = append(consumed, mark("consume", x))
	}
	fmt.Println(consumed, trace)
	trace = nil
	mapping := map[int]int{}
	for _, x := range []int{1, 2} {
		k := mark("key", 0)
		v := mark("value", x)
		mapping[k] = v
	}
	fmt.Println(mapping, trace)
	trace = nil
	xs := []int{1, 2, 3}
	copied := []int{}
	for _, x := range [3]int{1, 2, 3} {
		if x == 1 {
			xs[1] = 9
		}
		copied = append(copied, x)
	}
	fmt.Println(copied, xs)
	deferred := []int{}
	for _, x := range []int{7, 8} {
		deferred = append(deferred, mark("defer", x))
	}
	defer fmt.Println(deferred, trace)
	fmt.Println(false, trace)
}
