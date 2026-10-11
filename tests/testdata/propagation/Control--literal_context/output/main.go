package main

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
func run() (tgo3 error) {
	tgo4 := tgo3
	tgo5 := tgo4
	type S struct{ v int }
	tgo1, tgo2 := step("A")
	tgo4 = tgo2
	if tgo4 != tgo5 {
		return tgo4
	}
	tgo8 := ([]S{{tgo1}})[0]
	tgo6, tgo7 := step("B")
	tgo4 = tgo7
	if tgo4 != tgo5 {
		return tgo4
	}
	a := []S{tgo8, {tgo6}}
	fmt.Print(a)
	return nil
}
func main() {
	for _, failure := range []int{0, 1, 2} {
		visit = 0
		failAt = failure
		err := run()
		if err == nil {
			fmt.Print("S|")
		} else if err == sentinel {
			fmt.Print("E|")
		} else {
			panic(err)
		}
	}
}
