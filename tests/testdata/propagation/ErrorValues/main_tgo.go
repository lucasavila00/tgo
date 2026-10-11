package main

import tgo7 "fmt"

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

func direct() (tgo2 error) {
	var tgo3 error = tgo2
	{
		var tgo1 error = target(true)
		if tgo1 != nil {
			tgo3 = tgo1
			return tgo3
		}
	}
	panic("not reached")
}

func wrapped() (tgo5 error) {
	var tgo6 error = tgo5
	{
		var tgo4 error = target(false)
		if tgo4 != nil {
			tgo6 = tgo4
			tgo6 = tgo7.Errorf("%s: %w", "target", tgo6)
			return tgo6
		}
	}
	panic("not reached")
}

func shadow(nil, true, false int) (tgo9 error) {
	var tgo10 error = tgo9
	fmt.Print(nil, true, false)
	{
		var tgo8 error = target(0 != 0)
		var tgo11 error
		if tgo8 != tgo11 {
			tgo10 = tgo8
			return tgo10
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
