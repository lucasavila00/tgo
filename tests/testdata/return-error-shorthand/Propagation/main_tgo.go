package main

import fmt_tgo10 "fmt"

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
	var tgo3 int
	var tgo4 int
	defer done()
	{
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = read("L", false)
		if tgo2 != nil {
			return tgo3, tgo4, tgo2
		}
		var tgo5 int
		var tgo6 error
		tgo5, tgo6 = read("R", fail)
		if tgo6 != nil {
			return tgo3, tgo4, tgo6
		}
		return tgo1, tgo5, nil
	}
}

func wrapped(fail bool) (int, error) {
	var tgo9 int
	defer done()
	{
		var tgo7 int
		var tgo8 error
		tgo7, tgo8 = read("W", fail)
		if tgo8 != nil {
			return tgo9, fmt_tgo10.Errorf("read: %w", tgo8)
		}
		return tgo7, nil
	}
}

func tuple(fail bool) (int, string, error) {
	var tgo14 int
	var tgo15 string
	{
		var tgo11 int
		var tgo12 string
		var tgo13 error
		tgo11, tgo12, tgo13 = pair(fail)
		if tgo13 != nil {
			return tgo14, tgo15, tgo13
		}
		return tgo11, tgo12, nil
	}
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
