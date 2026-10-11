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

func consume[T any](f func(...T), n int) {
	fmt.Print("C", n)
	f()
}

func run[T any](f func(...T), failAt int) (tgo3 error) {
	type tgo7 = T
	var tgo4 error = tgo3
	var tgo5 error = tgo4

	{
		T := 1
		_ = T
		{
			var tgo6 func(...tgo7) = f
			var tgo1 int
			var tgo2 error
			tgo1, tgo2 = next(failAt)
			tgo4 = tgo2
			if tgo4 != tgo5 {
				return tgo4
			}
			consume(tgo6, tgo1)
		}
		{
			var tgo11 func(...tgo7) = f
			var tgo8 int
			var tgo9 error
			tgo8, tgo9 = next(failAt)
			tgo4 = tgo9
			if tgo4 != tgo5 {
				tgo4 = tgo10.Errorf("%s: %w", "next", tgo4)
				return tgo4
			}
			consume(tgo11, tgo8)
		}
	}
	return nil
}

func main() {
	for _, failAt := range []int{0, 1, 2} {
		visits = 0
		err := run(func(...int) { fmt.Print("F") }, failAt)
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
