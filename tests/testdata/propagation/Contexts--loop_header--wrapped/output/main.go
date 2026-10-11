package main

import tgo6 "fmt"

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
func predicate(name string) (bool, error) { _, err := step(name); return true, err }
func pair() (int, string, error)          { v, err := step("P"); return v, "x", err }
func only() error                         { _, err := step("O"); return err }
func identity[T any](v T) (T, error)      { _, err := step("F"); return v, err }
func run() (tgo3 error) {
	tgo4 := tgo3
	tgo5 := tgo4
	{
		tgo8, tgo9 := step("I")
		tgo4 = tgo9
		if tgo4 != tgo5 {
			tgo4 = tgo6.Errorf("%s: %w", "step", tgo4)
			return tgo4
		}
		tgo10 := 0 == 0
		for i := tgo8; ; {
			if tgo10 {
				tgo10 = 0 != 0
			} else {
				i++
			}
			tgo7 := i
			tgo1, tgo2 := step("C")
			tgo4 = tgo2
			if tgo4 != tgo5 {
				tgo4 = tgo6.Errorf("%s: %w", "step", tgo4)
				return tgo4
			}
			if !(tgo7 < tgo1+1) {
				break
			}
			{
				fmt.Print("b")
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
