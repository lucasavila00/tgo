package main

import (
	"errors"
	"fmt"
)

var events []string

func effect(label string, value int) int {
	events = append(events, label)
	return value
}

func done() { events = append(events, "D") }

func ordered(flag bool) (int, int, error) {
	defer done()
	left := effect("A", 1)
	var right int
	if flag {
		right = effect("B", 2)
	} else {
		right = effect("C", 3)
	}
	return left, right, nil
}

func early(err error) (int, int, error) {
	defer done()
	effect("A", 1)
	return 0, 0, err
}

func pair() (int, string) {
	return effect("P", 7), "pair"
}

func multiple() (any, any, error) {
	first, second := pair()
	return first, second, nil
}

func main() {
	failure := errors.New("failure")
	for _, flag := range []bool{true, false} {
		events = nil
		left, right, err := ordered(flag)
		fmt.Println(left, right, err, events)
	}
	events = nil
	left, right, err := early(failure)
	fmt.Println(left, right, err == failure, events)
	events = nil
	first, second, err := multiple()
	fmt.Println(first, second, err, events)
}
