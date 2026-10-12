package main

import (
	"errors"
	"fmt"
)

func fail(err error) (int, string, error) {
	var tgo1 int
	var tgo2 string
	return tgo1, tgo2, err
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
