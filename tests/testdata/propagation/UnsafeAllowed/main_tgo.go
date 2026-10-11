package main

import fmt_tgo8 "fmt"

import (
	"fmt"
	u "unsafe"
)

type utility struct{}

func (utility) Sizeof(value int) int { return value }
func next() (int, error)             { return 7, nil }
func run() error {
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = next()
	if tgo2 != nil {
		return tgo2
	}
	value := struct{ X int }{tgo1}
	fmt.Println(u.Sizeof(value) > 0, u.Alignof(value) > 0, u.Offsetof(value.X))
	unsafe := utility{}
	{
		var tgo5 func(value int) int = unsafe.Sizeof
		var tgo3 int
		var tgo4 error
		tgo3, tgo4 = next()
		if tgo4 != nil {
			return tgo4
		}
		fmt.Println(tgo5(tgo3))
	}
	{
		var tgo6 int
		var tgo7 error
		tgo6, tgo7 = next()
		if tgo7 != nil {
			return fmt_tgo8.Errorf("next: %w", tgo7)
		}
		var tgo11 int = min(tgo6, 8)
		var tgo9 int
		var tgo10 error
		tgo9, tgo10 = next()
		if tgo10 != nil {
			return tgo10
		}
		fmt.Println(tgo11, max(6, tgo9))
	}
	return nil
}
func main() { fmt.Println(run()) }
