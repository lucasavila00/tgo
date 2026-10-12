package main

import "fmt"

type Result struct{ N int }
type Error interface{ Error() string }

func nilShadow(nil int) (Result, error) {
	_ = nil
	var tgo2 error
	return Result{7}, tgo2
}

func typeShadow(err error) (Result, error) {
	var zero_tgo3 Result
	type Result string
	var value Result = "local"
	_ = value
	return zero_tgo3, err
}

func collision(tgo1 int) (int, error) {
	return tgo1, nil
}

func builtinShadow(err Error) (int, Error) {
	var zero_tgo4 int
	type error int
	_ = error(3)
	return zero_tgo4, err
}

func main() {
	value, err := nilShadow(3)
	fmt.Println(value, err)
	value, err = typeShadow(nil)
	fmt.Println(value, err)
	number, err := collision(9)
	fmt.Println(number, err)
	number, err = builtinShadow(nil)
	fmt.Println(number, err)
}
