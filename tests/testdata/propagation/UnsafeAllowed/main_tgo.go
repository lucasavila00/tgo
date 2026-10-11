package main

import fmt_tgo10 "fmt"

import (
	"fmt"
	u "unsafe"
)

type utility struct{}

func (utility) Sizeof(value int) int { return value }
func next() (int, error)             { return 7, nil }
func run() (tgo3 error) {
	var tgo4 error = tgo3
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = next()
	if tgo2 != nil {
		tgo4 = tgo2
		return tgo4
	}
	value := struct{ X int }{tgo1}
	fmt.Println(u.Sizeof(value) > 0, u.Alignof(value) > 0, u.Offsetof(value.X))
	unsafe := utility{}
	{
		var tgo7 func(value int) int = unsafe.Sizeof
		var tgo5 int
		var tgo6 error
		tgo5, tgo6 = next()
		if tgo6 != nil {
			tgo4 = tgo6
			return tgo4
		}
		fmt.Println(tgo7(tgo5))
	}
	{
		var tgo8 int
		var tgo9 error
		tgo8, tgo9 = next()
		if tgo9 != nil {
			tgo4 = tgo9
			tgo4 = fmt_tgo10.Errorf("%s: %w", "next", tgo4)
			return tgo4
		}
		var tgo13 int = min(tgo8, 8)
		var tgo11 int
		var tgo12 error
		tgo11, tgo12 = next()
		if tgo12 != nil {
			tgo4 = tgo12
			return tgo4
		}
		fmt.Println(tgo13, max(6, tgo11))
	}
	return nil
}
func main() { fmt.Println(run()) }
