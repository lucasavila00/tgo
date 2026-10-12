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
	defer done()
	left, err := read("L", false)
	if err != nil {
		return 0, 0, err
	}
	right, err := read("R", fail)
	if err != nil {
		return 0, 0, err
	}
	return left, right, nil
}

func wrapped(fail bool) (int, error) {
	defer done()
	value, err := read("W", fail)
	if err != nil {
		return 0, fmt.Errorf("read: %w", err)
	}
	return value, nil
}

func tuple(fail bool) (int, string, error) {
	value, text, err := pair(fail)
	if err != nil {
		return 0, "", err
	}
	return value, text, nil
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
