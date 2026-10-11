package main

import tgo10 "fmt"

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

func run[T any](f func(...T), channels chan (<-chan T), failAt int) (tgo3 error) {
	type tgo6 = T
	var tgo4 error = tgo3
	{
		T := 1
		_ = T
		{
			var tgo5 func(...tgo6) = f
			var tgo7 chan (<-chan tgo6) = channels
			var tgo1 int
			var tgo2 error
			tgo1, tgo2 = next(failAt)
			if tgo2 != nil {
				tgo4 = tgo2
				return tgo4
			}
			consume(tgo5, tgo7, tgo1)
		}
		{
			var tgo11 func(...tgo6) = f
			var tgo12 chan (<-chan tgo6) = channels
			var tgo8 int
			var tgo9 error
			tgo8, tgo9 = next(failAt)
			if tgo9 != nil {
				tgo4 = tgo9
				tgo4 = tgo10.Errorf("%s: %w", "next", tgo4)
				return tgo4
			}
			consume(tgo11, tgo12, tgo8)
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
