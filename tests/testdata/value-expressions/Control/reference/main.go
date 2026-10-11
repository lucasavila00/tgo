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

func headers() int {
	total := 0
	for i := 0; ; i++ {
		if i == 1 || i >= 3 {
			break
		}
		total += i
	}
	checks := 0
	for i := 0; ; i++ {
		checks++
		if checks == 1 {
			continue
		}
		if checks >= 3 {
			break
		}
		total += i
	}
	for i := 0; i < 3; {
		total += i
		if i == 1 {
			break
		}
		i++
	}
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < 5; i++ {
			if i == 1 {
				continue
			}
			if i == 3 {
				break
			}
			value := i
			if value >= 0 {
				total += value
			}
		}
	}
	return total
}

func labelledLoop() int {
	turns := 0
again:
	for i := func() int {
		if turns < 2 {
			turns++
		}
		return 0
	}(); i < 1; i++ {
		if turns < 2 {
			goto again
		}
		break again
	}
	return turns
}

func main() {
	if labelledLoop() != 2 {
		panic("loop label")
	}
	if headers() != 6 {
		panic("header jump target")
	}
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

func labelledValue() int { return 6 }
func tail() int {
	if true {
		goto value
	value:
		return labelledValue()
	}
	return 0
}
