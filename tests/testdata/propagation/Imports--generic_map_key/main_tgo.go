package main

import tgo15 "slices"
import tgo11 "iter"
import tgo9 "maps"

import (
	"fmt"

	"example.com/program/foreign"
)

func run[K comparable](key K) (tgo3 error) {
	tgo4 := tgo3
	tgo5 := tgo4
	tgo6 := key
	tgo1, tgo2 := foreign.Number("A")
	tgo4 = tgo2
	if tgo4 != tgo5 {
		return tgo4
	}
	tgo16 := tgo15.Collect(tgo9.Values(foreign.Generic[K]{tgo10(tgo9.Keys(foreign.Generic[K]{})): {Number: int(tgo1)}}))[0]
	tgo17 := key
	tgo7, tgo8 := foreign.Number("B")
	tgo4 = tgo8
	if tgo4 != tgo5 {
		return tgo4
	}
	v := foreign.Generic[K]{tgo6: tgo16, tgo17: {Number: int(tgo7)}}
	fmt.Print(len(v))
	return nil
}

func main() {
	for _, failure := range []int{0, 1, 2} {
		foreign.Reset(failure)
		err := run(7)
		if err == nil {
			fmt.Print("S|")
		} else if err == foreign.Failure {
			fmt.Print("E|")
		} else {
			panic(err)
		}
	}
}

func tgo10[tgo12 interface {
}](tgo13 tgo11.Seq[tgo12]) (tgo14 tgo12) {
	return
}
