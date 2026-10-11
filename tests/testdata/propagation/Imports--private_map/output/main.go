package main

import tgo15 "slices"
import tgo9 "iter"
import tgo7 "maps"

import (
	"errors"
	"example.com/program/foreign"
	"fmt"
)

func run() (tgo3 error) {
	tgo4 := tgo3
	tgo5 := tgo4
	tgo6 := tgo8(tgo7.Keys(foreign.Table{}))
	tgo6.ID = 1
	tgo1, tgo2 := foreign.Number("A")
	tgo4 = tgo2
	if tgo4 != tgo5 {
		return tgo4
	}
	tgo16 := tgo15.Collect(tgo7.Values(foreign.Table{tgo8(tgo7.Keys(foreign.Table{})): {Number: int(tgo1)}}))[0]
	tgo17 := tgo8(tgo7.Keys(foreign.Table{}))
	tgo17.ID = 2
	tgo13, tgo14 := foreign.Number("B")
	tgo4 = tgo14
	if tgo4 != tgo5 {
		return tgo4
	}
	v := foreign.Table{tgo6: tgo16, tgo17: {Number: int(tgo13)}}
	fmt.Print(len(v))
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

func tgo8[tgo10 interface {
}](tgo11 tgo9.Seq[tgo10]) (tgo12 tgo10) {
	return
}
