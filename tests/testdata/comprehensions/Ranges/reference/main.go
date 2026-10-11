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
	doubled := []int{}
	for _, x := range xs {
		doubled = append(doubled, x*2)
	}
	fmt.Println(doubled)
	array := []int{}
	for _, x := range [3]int{4, 5, 6} {
		array = append(array, x)
	}
	fmt.Println(array)
	indices := []int{}
	for i := range (*[3]int)(nil) {
		indices = append(indices, i)
	}
	fmt.Println(indices)
	runes := []rune{}
	for _, r := range "aé" {
		runes = append(runes, r)
	}
	fmt.Println(runes)
	mapping := map[int]string{}
	for i, value := range pairs {
		mapping[i] = value
	}
	fmt.Println(mapping)
	filtered := []int{}
	for x := range sequence {
		if x < 5 {
			filtered = append(filtered, x)
		}
	}
	fmt.Println(filtered)
	stopped := []int{}
	for x := range sequence {
		if x > 2 {
			break
		}
		stopped = append(stopped, x)
	}
	fmt.Println(stopped)
	integers := []int{}
	for i := range 3 {
		integers = append(integers, i)
	}
	fmt.Println(integers)
	visitsResult := []int{}
	for range visits {
		visitsResult = append(visitsResult, 1)
	}
	fmt.Println(visitsResult)
	channel := make(chan int, 2)
	channel <- 8
	channel <- 9
	close(channel)
	received := []int{}
	for x := range channel {
		received = append(received, x)
	}
	fmt.Println(received)
	strings := map[string]int{}
	for key, value := range map[string]int{"a": 1, "b": 2} {
		strings[key] = value * 2
	}
	fmt.Println(strings)
	tripled := map[int]int{}
	for i, x := range xs {
		tripled[i] = x * 3
	}
	fmt.Println(tripled)
	products := [][2]int{}
	for _, x := range xs {
		if x > 1 {
			for y := range x {
				products = append(products, [2]int{x, y})
			}
		}
	}
	fmt.Println(products)
	var absent []int
	empty := []int{}
	for _, x := range absent {
		empty = append(empty, x)
	}
	fmt.Println(empty == nil, len(empty))
	closures := []func() int{}
	for _, x := range xs {
		closures = append(closures, func() int { return x })
	}
	fmt.Println(closures[0](), closures[1](), closures[2]())
}
