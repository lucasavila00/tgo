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
	i := 0
again:
	for i < 2 {
		i++
		{
			var tgo1 int
			var tgo2 error
			tgo1, tgo2 = step("B")
			if tgo2 != nil {
				return tgo2
			}
			_ = tgo1
		}
		if i == 1 {
			continue again
		}
		break again
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
