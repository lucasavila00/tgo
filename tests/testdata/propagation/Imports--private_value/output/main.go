package main

import (
	"errors"
	"example.com/program/foreign"
	"fmt"
)

func run() (tgo3 error) {
	tgo4 := tgo3
	tgo5 := tgo4
	tgo1, tgo2 := foreign.Number("A")
	tgo4 = tgo2
	if tgo4 != tgo5 {
		return tgo4
	}
	v := tgo1
	fmt.Printf("%T", v)
	return nil
}
func main() {
	for _, failure := range []int{0, 1, 2} {
		foreign.Reset(failure)
		err := run()
		if err == nil {
			fmt.Print("S|")
		} else if errors.Is(err, foreign.Failure) {
			if err != foreign.Failure {
				fmt.Print(err.Error())
			}
			fmt.Print("E|")
		} else {
			panic(err)
		}
	}
}
