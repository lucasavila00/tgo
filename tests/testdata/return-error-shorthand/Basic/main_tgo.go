package main

import (
	"errors"
	"fmt"
)

func fail(err error) (int, string, error) {
	var zero_tgo1 int
	var zero_tgo2 string
	return zero_tgo1, zero_tgo2, err
}

func succeed() (int, string, error) {
	return 7, "ok", nil
}

func main() {
	failure := errors.New("failed")
	value, text, err := fail(failure)
	fmt.Println(value, text == "", err == failure)
	value, text, err = succeed()
	fmt.Println(value, text, err)
	value, err = one()
	fmt.Println(value, err)
}
