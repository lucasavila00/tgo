package main

import tgo9 "fmt"

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
	var tgo4 error = tgo3
	{
		var tgo1 error = target(true)
		tgo3 = tgo1
		if tgo3 != tgo4 {
			return tgo3
		}
	}
	panic("not reached")
}

func wrapped() (tgo6 error) {
	var tgo7 error = tgo6
	var tgo8 error = tgo7
	{
		var tgo5 error = target(false)
		tgo7 = tgo5
		if tgo7 != tgo8 {
			tgo7 = tgo9.Errorf("%s: %w", "target", tgo7)
			return tgo7
		}
	}
	panic("not reached")
}

func shadow(nil, true, false int) (tgo11 error) {
	var tgo12 error = tgo11
	var tgo13 error = tgo12

	fmt.Print(nil, true, false)
	{
		var tgo10 error = target(0 != 0)
		tgo12 = tgo10
		if tgo12 != tgo13 {
			return tgo12
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
