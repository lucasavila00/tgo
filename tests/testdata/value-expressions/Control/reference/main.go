package main

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
	defer fmt.Print("D")
	var value int
	if wrap {
		result, err := next(fail)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", "next", err)
		}
		value = result
	} else {
		result, err := next(fail)
		if err != nil {
			return 0, err
		}
		value = result
	}
	return value, nil
}

func loop() int {
	total := 0
	for i := 0; i < 5; i++ {
		if i == 1 {
			fmt.Print("C")
			continue
		}
		if i == 3 {
			fmt.Print("B")
			break
		}
		total += i
	}
	return total
}

func jump(skip bool) int {
	value := 0
	if skip {
		goto done
	}
	value = 9
done:
	return value
}

func deferred() int {
	if true {
		defer fmt.Print("F")
		return 5
	}
	return 0
}

func main() {
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
