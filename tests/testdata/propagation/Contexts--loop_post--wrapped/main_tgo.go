package main

import tgo5 "fmt"

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("failure")
var visit, failAt int

func step(name string) (int, error) {
	fmt.Print(name)
	visit++
	if visit == failAt {
		return 0, sentinel
	}
	return 1, nil
}

func predicate(name string) (bool, error) {
	_, err := step(name)
	return true, err
}

func pair() (int, string, error) {
	v, err := step("P")
	return v, "x", err
}

func only() error {
	_, err := step("O")
	return err
}

func identity[T any](v T) (T, error) {
	_, err := step("F")
	return v, err
}

func run() (tgo3 error) {
	var tgo4 error = tgo3
	{
		var tgo6 bool = 0 == 0

		for i := 0; ; {
			if tgo6 {
				tgo6 = 0 != 0
			} else {
				var tgo1 int
				var tgo2 error
				tgo1, tgo2 = step("P")
				if tgo2 != nil {
					tgo4 = tgo2
					tgo4 = tgo5.Errorf("%s: %w", "step", tgo4)
					return tgo4
				}
				i += tgo1
			}
			if !(i < 2) {
				break
			}
			{
				fmt.Print("b")
				continue
			}
		}
	}
	return nil
}

func main() {
	for _, failure := range []int{0, 1, 2} {
		visit = 0
		failAt = failure
		err := run()
		if err == nil {
			fmt.Print("S|")
		} else if errors.Unwrap(err) == sentinel {
			fmt.Print("E|")
		} else {
			panic(err)
		}
	}
}
