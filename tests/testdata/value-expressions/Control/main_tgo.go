package main

import fmt_tgo5 "fmt"

import (
	"errors"
	"fmt"
)

var failure = errors.New("failure")

func next(fail bool) (int, error) {
	fmt.Print("N")
	if fail {
		return 0, failure
	}
	return 7, nil
}

func checked(wrap, fail bool) (int, error) {
	var tgo4 int
	defer fmt.Print("D")
	var tgo1 int
	if wrap {
		var tgo2 int
		var tgo3 error
		tgo2, tgo3 = next(fail)
		if tgo3 != nil {
			return tgo4, fmt_tgo5.Errorf("%s: %w", "next", tgo3)
		}
		tgo1 = tgo2
	} else {
		var tgo6 int
		var tgo7 error
		tgo6, tgo7 = next(fail)
		if tgo7 != nil {
			return tgo4, tgo7
		}
		tgo1 = tgo6
	}
	value := tgo1

	return value, nil
}

func loop() int {
	total := 0
	for i := 0; i < 5; i++ {
		var tgo8 int
		if i == 1 {
			fmt.Print("C")
			continue
		} else if i == 3 {
			fmt.Print("B")
			break
		} else {
			tgo8 = i
		}
		value := tgo8

		total += value
	}
	return total
}

func jump(skip bool) int {
	value := 0
	{
		var tgo9 int
		if skip {
			goto done
		} else {
			tgo9 = 9
		}
		value = tgo9
	}

done:
	return value
}

func labelledValue() int { return 6 }

func tail() int {
	{
		var tgo10 int
		if true {
			goto value
		value:
			tgo10 = labelledValue()
		} else {
			tgo10 = 0
		}
		return tgo10
	}

}

func deferred() int {
	{
		var tgo11 int
		if true {
			defer fmt.Print("F")
			tgo11 = 5
		} else {
			tgo11 = 0
		}
		return tgo11
	}

}

func main() {
	if tail() != 6 {
		panic("labelled value")
	}
	value, err := checked(false, false)
	if value != 7 || err != nil {
		panic("success")
	}
	fmt.Print(";")
	value, err = checked(false, true)
	if value != 0 || err != failure {
		panic("direct error identity")
	}
	fmt.Print(";")
	value, err = checked(true, true)
	if value != 0 || !errors.Is(err, failure) || err.Error() != "next: failure" {
		panic("wrapped error")
	}
	fmt.Print(";")
	fmt.Print("loop=", loop(), ";jump=", jump(true), ",", jump(false), ";defer=", deferred(), "\n")
}
