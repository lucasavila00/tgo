package main

import (
	"errors"
	"fmt"
)

var trace []string
var failure = errors.New("failed")

type payload struct {
	Left  int
	Right int
}

func mark(name string, value int) int { trace = append(trace, name); return value }
func read(fail bool) (int, error) {
	trace = append(trace, "read")
	if fail {
		return 0, failure
	}
	return 4, nil
}
func build(fail bool) (payload, error) {
	right := mark("right", 2)
	left, err := read(fail)
	if err != nil {
		return payload{}, err
	}
	return payload{Left: left, Right: right}, nil
}
func main() {
	value := payload{Right: mark("right", 2), Left: mark("left", 1)}
	fmt.Println(value, trace)
	trace = nil
	pointer := &payload{Right: mark("address-right", 3), Left: mark("address-left", 2)}
	fmt.Println(*pointer, trace)
	trace = nil
	value = payload{Left: mark("if", 5), Right: mark("switch", 6)}
	fmt.Println(value, trace)
	trace = nil
	trace = append(trace, "pick")
	fmt.Println(value.Left, value.Right, trace)
	trace = nil
	good, err := build(false)
	fmt.Println(good, err, trace)
	trace = nil
	_, err = build(true)
	fmt.Println(err == failure, trace)
}
