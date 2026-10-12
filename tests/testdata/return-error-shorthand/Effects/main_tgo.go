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
	{
		var tgo2 int = effect("A", 1)
		var tgo1 int
		if flag {
			tgo1 = effect("B", 2)
		} else {
			tgo1 = effect("C", 3)
		}
		return tgo2, tgo1, nil
	}
}

func early(err error) (int, int, error) {
	var tgo4 int
	var tgo5 int
	defer done()
	{
		var tgo6 int = effect("A", 1)
		var tgo3 int
		if true {
			return tgo4, tgo5, err
			_ = 2 == 2
		} else {
			tgo3 = 3
		}
		return tgo6, tgo3, nil
	}
}

func pair() (int, string) {
	return effect("P", 7), "pair"
}

func multiple() (any, any, error) {
	{
		var tgo7 any
		var tgo8 any
		tgo7, tgo8 = pair()
		return tgo7, tgo8, nil
	}
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
