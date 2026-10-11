package main

import tgo8 "fmt"

import (
	"errors"
	"example.com/program/foreign"
	"fmt"
)

func run() (tgo3 error) {
	tgo4 := tgo3
	tgo5 := tgo4
	{
		tgo1, tgo2 := foreign.Factory()
		tgo4 = tgo2
		if tgo4 != tgo5 {
			return tgo4
		}
		tgo6, tgo7 := tgo1()
		tgo4 = tgo7
		if tgo4 != tgo5 {
			tgo4 = tgo8.Errorf("%s: %w", "foreign.Factory()", tgo4)
			return tgo4
		}
		_ = tgo6
	}
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
