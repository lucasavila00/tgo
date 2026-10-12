package main

import (
	"fmt"
	u "unsafe"
)

type utility struct{}

func (utility) Sizeof(value int) int { return value }
func next() (int, error)             { return 7, nil }
func run() error {
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = next()
	if error_tgo2 != nil {
		return error_tgo2
	}
	value := struct{ X int }{value_tgo1}
	fmt.Println(u.Sizeof(value) > 0, u.Alignof(value) > 0, u.Offsetof(value.X))
	unsafe := utility{}
	var value_tgo5 func(value int) int = unsafe.Sizeof
	var value_tgo3 int
	var error_tgo4 error
	value_tgo3, error_tgo4 = next()
	if error_tgo4 != nil {
		return error_tgo4
	}
	fmt.Println(value_tgo5(value_tgo3))
	var value_tgo6 int
	var error_tgo7 error
	value_tgo6, error_tgo7 = next()
	if error_tgo7 != nil {
		return fmt.Errorf("next: %w", error_tgo7)
	}
	var value_tgo10 int = min(value_tgo6, 8)
	var value_tgo8 int
	var error_tgo9 error
	value_tgo8, error_tgo9 = next()
	if error_tgo9 != nil {
		return error_tgo9
	}
	fmt.Println(value_tgo10, max(6, value_tgo8))
	return nil
}
func main() { fmt.Println(run()) }
