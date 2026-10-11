package main

import "fmt"

func main() {
	xs := []int{1, 2, 3}
	values := []int{}
	for _, x := range xs {
		values = append(values, x*2)
	}
	fmt.Println(values, "append", "make", "len")
	mapping := map[int]int{}
	for i, x := range xs {
		mapping[i] = x
	}
	fmt.Println(mapping)
}
