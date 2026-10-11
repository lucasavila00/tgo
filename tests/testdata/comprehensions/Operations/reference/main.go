package main

import "fmt"

var trace []string

func mark(label string, value int) int { trace = append(trace, label); return value }
func mapPanic() {
	defer func() { fmt.Println(recover() != nil, trace); trace = nil }()
	result := map[any]int{}
	for x := range 1 {
		var key any = []int{x}
		value := mark("value before hash", x)
		result[key] = value
	}
}
func deferred() {
	values := []int{}
	for x := range 3 {
		defer fmt.Println("visit", x)
		values = append(values, x)
	}
	fmt.Println(values)
}
func main() {
	mapPanic()
	deferred()
	source := [3]int{1, 2, 3}
	mapping := map[string]int{"a": 1}
	retained := map[string]int{}
	for key, value := range mapping {
		delete(mapping, key)
		retained[key] = value
	}
	fmt.Println(retained, mapping)
	copied := []int{}
	for _, x := range source {
		if x == 1 {
			source[1] = 9
		}
		copied = append(copied, x)
	}
	fmt.Println(copied, source)
	outer := []int{1, 2}
	shadowed := []int{}
	for _, outer := range outer {
		shadowed = append(shadowed, outer)
	}
	fmt.Println(shadowed)
	stores := [1][]int{}
	index := mark("destination", 0)
	values := []int{}
	for x := range 2 {
		values = append(values, mark("output", x))
	}
	stores[index] = values
	fmt.Println(stores, trace)
	trace = nil
	channel := make(chan string, 1)
	values = []int{}
	for x := range 2 {
		values = append(values, mark("go argument", x))
	}
	go func(values []int) { channel <- fmt.Sprint(values) }(values)
	fmt.Println(<-channel, trace)
	trace = nil
	ready := make(chan int, 1)
	ready <- 7
	channels := []chan int{}
	for range 1 {
		trace = append(trace, "select operand")
		channels = append(channels, ready)
	}
	select {
	case value := <-channels[0]:
		fmt.Println(value, trace)
	}
}
