package main

import tgo8 "fmt"

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
		{
			var tgo3 func(...tgo4) = f
			var tgo5 chan (<-chan tgo4) = channels
			var tgo1 int
			var tgo2 error
			tgo1, tgo2 = next(failAt)
			if tgo2 != nil {
				return tgo2
			}
			consume(tgo3, tgo5, tgo1)
		}
		{
			var tgo9 func(...tgo4) = f
			var tgo10 chan (<-chan tgo4) = channels
			var tgo6 int
			var tgo7 error
			tgo6, tgo7 = next(failAt)
			if tgo7 != nil {
				return tgo8.Errorf("%s: %w", "next", tgo7)
			}
			consume(tgo9, tgo10, tgo6)
		}
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
