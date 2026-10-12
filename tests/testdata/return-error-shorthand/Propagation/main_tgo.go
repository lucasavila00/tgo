package main

import (
	"errors"
	"fmt"
)

var failure = errors.New("failure")
var events []string

func read(label string, fail bool) (int, error) {
	events = append(events, label)
	if fail {
		return 99, failure
	}
	return 7, nil
}

func pair(fail bool) (int, string, error) {
	events = append(events, "P")
	if fail {
		return 99, "discard", failure
	}
	return 8, "pair", nil
}

func done() { events = append(events, "D") }

func original(fail bool) (int, int, error) {
	var zero_tgo3 int
	var zero_tgo4 int
	defer done()
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = read("L", false)
	if error_tgo2 != nil {
		return zero_tgo3, zero_tgo4, error_tgo2
	}
	var value_tgo5 int
	var error_tgo6 error
	value_tgo5, error_tgo6 = read("R", fail)
	if error_tgo6 != nil {
		return zero_tgo3, zero_tgo4, error_tgo6
	}
	return value_tgo1, value_tgo5, nil
}

func wrapped(fail bool) (int, error) {
	var zero_tgo9 int
	defer done()
	var value_tgo7 int
	var error_tgo8 error
	value_tgo7, error_tgo8 = read("W", fail)
	if error_tgo8 != nil {
		return zero_tgo9, fmt.Errorf("read: %w", error_tgo8)
	}
	return value_tgo7, nil
}

func tuple(fail bool) (int, string, error) {
	var zero_tgo13 int
	var zero_tgo14 string
	var value_tgo10 int
	var value_tgo11 string
	var error_tgo12 error
	value_tgo10, value_tgo11, error_tgo12 = pair(fail)
	if error_tgo12 != nil {
		return zero_tgo13, zero_tgo14, error_tgo12
	}
	return value_tgo10, value_tgo11, nil
}

func main() {
	for _, fail := range []bool{false, true} {
		events = nil
		left, right, err := original(fail)
		fmt.Println(left, right, err == failure, errors.Is(err, failure), events)
		events = nil
		left, err = wrapped(fail)
		fmt.Println(left, err == failure, errors.Is(err, failure), events)
		events = nil
		value, text, err := tuple(fail)
		fmt.Println(value, text == "pair", err == failure, events)
	}
}
