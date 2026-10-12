package main

import (
	"errors"
	"fmt"
)

type Failure struct{}

func (*Failure) Error() string {
	return "failure"
}

var sentinel = &Failure{}
var any = 7

func target(typedNil bool) error {
	if typedNil {
		var p *Failure
		return p
	}
	return sentinel
}

func direct() error {
	var error_tgo1 error = target(true)
	if error_tgo1 != nil {
		return error_tgo1
	}
	panic("not reached")
}

func wrapped() error {
	var error_tgo2 error = target(false)
	if error_tgo2 != nil {
		return fmt.Errorf("target: %w", error_tgo2)
	}
	panic("not reached")
}

func shadow(nil, true, false int) error {
	fmt.Print(nil, true, false)
	var error_tgo3 error = target(0 != 0)
	var zero_tgo4 error
	if error_tgo3 != zero_tgo4 {
		return error_tgo3
	}
	var success error
	return success
}

func main() {
	var p *Failure
	e := direct()
	fmt.Print(e == p, "|")
	e = wrapped()
	var caught *Failure
	fmt.Print(e.Error(), "|", errors.Is(e, sentinel), "|", errors.As(e, &caught), caught == sentinel, "|")
	fmt.Print(shadow(0, 1, 2) == sentinel)
}
