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

func run() error {
	done := make(chan struct{})
	var value_tgo3 func(v int) = func(v int) {
		fmt.Print("g")
		close(done)
	}
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = step("G")
	if error_tgo2 != nil {
		return error_tgo2
	}
	go value_tgo3(value_tgo1)
	<-done
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
