package main

import "fmt"

func effect(name string, value int) int {
	fmt.Print(name)
	return value
}

func pair(value int) (int, int) { return value, value + 1 }

var selected = func() int {
	if ready {
		value := later
		return effect("Y", value)
	}
	return effect("X", 0)
}()

var first, second = pair(func() int {
	if ready {
		return selected
	}
	return 0
}())
var later = effect("L", 7)
var ready = true

func main() {
	fmt.Println(selected, first, second)
}
