package main

import tgo11 "fmt"

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
	type tgo7 = T
	var tgo4 error = tgo3
	var tgo5 error = tgo4

	{
		T := 1
		_ = T
		{
			var tgo6 func(...tgo7) = f
			var tgo8 chan (<-chan tgo7) = channels
			var tgo1 int
			var tgo2 error
			tgo1, tgo2 = next(failAt)
			tgo4 = tgo2
			if tgo4 != tgo5 {
				return tgo4
			}
			consume(tgo6, tgo8, tgo1)
		}
		{
			var tgo12 func(...tgo7) = f
			var tgo13 chan (<-chan tgo7) = channels
			var tgo9 int
			var tgo10 error
			tgo9, tgo10 = next(failAt)
			tgo4 = tgo10
			if tgo4 != tgo5 {
				tgo4 = tgo11.Errorf("%s: %w", "next", tgo4)
				return tgo4
			}
			consume(tgo12, tgo13, tgo9)
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
