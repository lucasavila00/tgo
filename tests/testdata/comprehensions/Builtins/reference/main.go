package main

import (
	"fmt"
	"unsafe"
)

var effects int

func main() {
	values := []int{}
	for i := range 2 {
		effects++
		values = append(values, i)
	}
	fmt.Println(len([1][]int{values}), effects)
	values = []int{}
	for i := range 2 {
		effects++
		values = append(values, i)
	}
	fmt.Println(cap([1][]int{values}), effects)
	values = []int{}
	for i := range 2 {
		effects++
		values = append(values, i)
	}
	for range [1][]int{values} {
		fmt.Println("range", effects)
	}
	fmt.Println(unsafe.Sizeof([]int{}), effects)
	fmt.Println(unsafe.Alignof(map[int]int{}), effects)
	fmt.Println(unsafe.Offsetof(struct{ Values []int }{}.Values), effects)
	indices := []int{}
	for i := range [2]int{1, 2} {
		indices = append(indices, i)
	}
	fmt.Println(indices)
	var pointer *[2]int
	indices = []int{}
	for i := range *pointer {
		indices = append(indices, i)
	}
	fmt.Println(indices)
}
