package main

import fmt_tgo3 "fmt"

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
	{
		var tgo1 error = target(true)
		if tgo1 != nil {
			return tgo1
		}
	}
	panic("not reached")
}

func wrapped() error {
	{
		var tgo2 error = target(false)
		if tgo2 != nil {
			return fmt_tgo3.Errorf("target: %w", tgo2)
		}
	}
	panic("not reached")
}

func shadow(nil, true, false int) error {
	fmt.Print(nil, true, false)
	{
		var tgo4 error = target(0 != 0)
		var tgo5 error
		if tgo4 != tgo5 {
			return tgo4
		}
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
