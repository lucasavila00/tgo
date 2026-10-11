package main

import tgo11 "fmt"

import (
	"fmt"
	u "unsafe"
)

type utility struct{}

func (utility) Sizeof(value int) int { return value }
func next() (int, error)             { return 7, nil }
func run() (tgo3 error) {
	tgo4 := tgo3
	tgo5 := tgo4
	tgo1, tgo2 := next()
	tgo4 = tgo2
	if tgo4 != tgo5 {
		return tgo4
	}
	value := struct{ X int }{tgo1}
	fmt.Println(u.Sizeof(value) > 0, u.Alignof(value) > 0, u.Offsetof(value.X))
	unsafe := utility{}
	{
		tgo8 := unsafe.Sizeof
		tgo6, tgo7 := next()
		tgo4 = tgo7
		if tgo4 != tgo5 {
			return tgo4
		}
		fmt.Println(tgo8(tgo6))
	}
	{
		tgo9, tgo10 := next()
		tgo4 = tgo10
		if tgo4 != tgo5 {
			tgo4 = tgo11.Errorf("%s: %w", "next", tgo4)
			return tgo4
		}
		tgo14 := min(tgo9, 8)
		tgo12, tgo13 := next()
		tgo4 = tgo13
		if tgo4 != tgo5 {
			return tgo4
		}
		fmt.Println(tgo14, max(6, tgo12))
	}
	return nil
}
func main() { fmt.Println(run()) }
