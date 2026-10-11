package main

import "fmt"

func effect(name string, value int) int {
	fmt.Print(name)
	return value
}

func pair(value int) (int, int) { return value, value + 1 }

var selected = func() int {
	var tgo1 int
	if ready {
		value := later
		tgo1 = effect("Y", value)
	} else {
		tgo1 = effect("X", 0)
	}
	return tgo1
}()

var first, second = func() (int, int) {
	var tgo2 int
	if ready {
		tgo2 = selected
	} else {
		tgo2 = 0
	}
	return pair(tgo2)
}()
var later = effect("L", 7)
var ready = true

func main() {
	fmt.Println(selected, first, second)
}
