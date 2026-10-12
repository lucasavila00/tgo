package main

import fmt_tgo8 "fmt"

import (
	"errors"
	"fmt"
)

type failure struct{}

func (*failure) Error() string { return "failure" }

var sentinel = &failure{}

func target(fail bool) (int, string, *failure) {
	fmt.Print("T")
	if fail {
		return 1, "value", sentinel
	}
	return 1, "value", nil
}

func discarded(fail bool) error {
	var error_tgo2 *failure
	_, _, error_tgo2 = target(fail)
	if error_tgo2 != nil {
		return fmt.Errorf("target: %w", error_tgo2)
	}
	return nil
}

func saved() error {
	value := 10
	capture := func() int { return value }
	var value_tgo3 int
	var value_tgo4 string
	var error_tgo5 *failure
	value_tgo3, value_tgo4, error_tgo5 = target(false)
	if error_tgo5 != nil {
		return error_tgo5
	}
	value, text := value_tgo3, value_tgo4
	fmt.Print(value, text, capture())
	return nil
}

func blank() error {
	var error_tgo6 *failure
	_, _, error_tgo6 = target(false)
	if error_tgo6 != nil {
		return error_tgo6
	}
	return nil
}

func shadowed() error {
	fmt := "hidden"
	_ = fmt
	var error_tgo7 *failure
	_, _, error_tgo7 = target(true)
	if error_tgo7 != nil {
		return fmt_tgo8.Errorf("target: %w", error_tgo7)
	}
	return nil
}

func laterShadow() error {
	var error_tgo9 *failure
	_, _, error_tgo9 = target(false)
	if error_tgo9 != nil {
		return fmt.Errorf("target: %w", error_tgo9)
	}
	fmt := "hidden"
	_ = fmt
	var error_tgo10 *failure
	_, _, error_tgo10 = target(true)
	if error_tgo10 != nil {
		return fmt_tgo8.Errorf("target: %w", error_tgo10)
	}
	return nil
}

func jumped() error {
	goto after
	{
		var error_tgo11 *failure
		_, _, error_tgo11 = target(true)
		if error_tgo11 != nil {
			return fmt.Errorf("target: %w", error_tgo11)
		}
	}
after:
	{
		var error_tgo12 *failure
		_, _, error_tgo12 = target(false)
		if error_tgo12 != nil {
			return error_tgo12
		}
	}
	return nil
}

func closure() error {
	fmt := "hidden"
	_ = fmt
	call := func() error {
		var error_tgo13 *failure
		_, _, error_tgo13 = target(true)
		if error_tgo13 != nil {
			return fmt_tgo8.Errorf("target: %w", error_tgo13)
		}
		return nil
	}
	return call()
}

func main() {
	fmt.Print(discarded(false) == nil, "|")
	err := discarded(true)
	fmt.Print(errors.Is(err, sentinel), "|", err.Error(), "|")
	fmt.Print(saved() == nil, "|")
	fmt.Print(blank() == nil, "|")
	fmt.Print(errors.Is(shadowed(), sentinel), "|")
	fmt.Print(errors.Is(laterShadow(), sentinel), "|")
	fmt.Print(jumped() == nil, "|")
	fmt.Print(errors.Is(closure(), sentinel), "|")
	fmt.Print(errors.Is(aliased(), sentinel))
}
