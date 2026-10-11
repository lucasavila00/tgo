package main

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("failure")
var visit, failAt int

type N uint64

func step(name string) (int, error) {
	fmt.Print(name)
	visit++
	if visit == failAt {
		return 0, sentinel
	}
	return 1, nil
}

func run() error {
	defer func() {
		if recover() != nil {
			fmt.Print("r")
		}
	}()
	var a *[2]N
	for a[tgoPropagation1(step("I"))] = range 0 {

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
		} else if err == sentinel {
			fmt.Print("E|")
		} else {
			panic(err)
		}
	}
}
func tgoPropagation1[T0 interface {
}, T1 interface {
}](v0 T0, err T1) T0 {
	return v0
}
