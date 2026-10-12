package main

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("failure")
var visits int

func next(failAt int) (int, error) {
	fmt.Print("N")
	visits++
	if visits == failAt {
		return 0, sentinel
	}
	return visits, nil
}

func consume[T any](f func(...T), channels chan (<-chan T), n int) {
	inner := <-channels
	value := <-inner
	fmt.Print("C", n, ":", value)
	f(value)
}

func run[T any](f func(...T), channels chan (<-chan T), failAt int) error {
	type tgo4 = T
	{
		T := 1
		_ = T
		var value_tgo3 func(...tgo4) = f
		var value_tgo5 chan (<-chan tgo4) = channels
		var value_tgo1 int
		var error_tgo2 error
		value_tgo1, error_tgo2 = next(failAt)
		if error_tgo2 != nil {
			return error_tgo2
		}
		consume(value_tgo3, value_tgo5, value_tgo1)
		var value_tgo8 func(...tgo4) = f
		var value_tgo9 chan (<-chan tgo4) = channels
		var value_tgo6 int
		var error_tgo7 error
		value_tgo6, error_tgo7 = next(failAt)
		if error_tgo7 != nil {
			return fmt.Errorf("next: %w", error_tgo7)
		}
		consume(value_tgo8, value_tgo9, value_tgo6)
	}
	return nil
}

func main() {
	for _, failAt := range []int{0, 1, 2} {
		visits = 0
		inner := make(chan int, 2)
		inner <- 7
		inner <- 8
		channels := make(chan (<-chan int), 2)
		channels <- inner
		channels <- inner
		err := run(func(...int) { fmt.Print("F") }, channels, failAt)
		switch {
		case err == nil:
			fmt.Print("S|")
		case err == sentinel:
			fmt.Print("E|")
		case errors.Is(err, sentinel):
			fmt.Print("W|")
		default:
			panic(err)
		}
	}
}
