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
	tgo8 := (foreign.List{{Number: int(tgo1)}})[0]
	tgo6, tgo7 := foreign.Number("B")
	tgo4 = tgo7
	if tgo4 != tgo5 {
		return tgo4
	}
	v := foreign.List{tgo8, {Number: int(tgo6)}}
	fmt.Print(v)
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
