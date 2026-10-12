package main

import (
	"errors"
	"fmt"
)

func count(values ...int) int { return len(values) }

func decimal(values []int) (float64, error) {
	return float64(count(values...)) + 1.5, nil // trailing comment
}

func newline(err error) (int, error) {
	var zero_tgo1 int
	return zero_tgo1, // leading comment
		err // operand comment
}

func raw() (string, error) {
	return `.. is text`, nil
}

func last() (int, error) {
	return 3, nil /* multiline
	comment */
}

func main() {
	value, err := decimal([]int{1, 2})
	fmt.Println(value, err)
	failure := errors.New("failure")
	number, err := newline(failure)
	fmt.Println(number, err == failure)
	text, err := raw()
	fmt.Println(text, err)
	number, err = last()
	fmt.Println(number, err)
}
