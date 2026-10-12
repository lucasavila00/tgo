package main

import "fmt"

type small int8

func pick[T any](x, y T) T                 { return x }
func identity[T any](x T) T                { return x }
func use[T any](x T, fn func(int) int) int { return fn(6) }
func exits(n int) int {
	var tgo1 int
	switch n {
	case 0:
		return 10
	default:
		return 20
	}
	var x int = tgo1
	return x
}

func main() {
	var tgo2 small
	switch 1 {
	case 1:
		tgo2 = 127
	default:
		tgo2 = 0
	}
	var n small = tgo2
	fmt.Println(n, n+1)
	var tgo3 *int
	switch 1 {
	case 1:
		tgo3 = nil
	default:
		tgo3 = nil
	}
	var p *int = tgo3
	fmt.Println(p == nil)
	var tgo4 any
	switch 1 {
	case 1:
		tgo4 = "text"
	default:
		tgo4 = 3
	}
	var a any = tgo4
	fmt.Println(a)
	var tgo5 any
	switch 1 {
	case 1:
		tgo5 = 4
	default:
		tgo5 = "text"
	}
	fmt.Println(pick(tgo5, any(5)))
	var tgo6 func(int) int
	switch 1 {
	case 1:
		tgo6 = identity
	default:
		tgo6 = identity
	}
	fmt.Println(use(1, tgo6))
	fmt.Println(exits(0), exits(1))
	var tgo7 any
	switch 1 {
	case 1:
		switch 2 {
		case 2:
			tgo7 = 7
		default:
			tgo7 = 8
		}
	default:
		tgo7 = 9
	}
	fmt.Println(tgo7)
	var tgo8 any
	if true {
		switch 2 {
		case 2:
			tgo8 = 8
		default:
			tgo8 = 9
		}
	} else {
		tgo8 = 10
	}
	fmt.Println(tgo8)
	xs := []int{0, 0}
	var value_tgo9 []int = xs
	var tgo10 int
	switch 1 {
	case 1:
		tgo10 = 0
	default:
		tgo10 = 1
	}
	var value_tgo11 []int = xs
	var tgo12 int
	switch 1 {
	case 1:
		tgo12 = 11
	default:
		tgo12 = 0
	}
	var tgo13 int
	switch 1 {
	case 1:
		tgo13 = 12
	default:
		tgo13 = 0
	}
	value_tgo9[tgo10], value_tgo11[1] = tgo12, tgo13
	fmt.Println(xs)
	ch := make(chan int, 1)
	var value_tgo14 chan int = ch
	var tgo15 int
	switch 1 {
	case 1:
		tgo15 = 13
	default:
		tgo15 = 0
	}
	select {
	case value_tgo14 <- tgo15:
	}
	fmt.Println(<-ch)
	i := 0
	var tgo16 []int
	switch 1 {
	case 1:
		tgo16 = []int{1, 2}
	default:
		tgo16 = nil
	}
	for range tgo16 {
		i++
	}
	fmt.Println(i)
	switch 1 {
	case 1:
		fmt.Print("ordinary ")
		fallthrough
	default:
		fmt.Println("fallthrough")
	}
	for {
		switch 1 {
		case 1:
			break
		}
		fmt.Println("ordinary break")
		break
	}
}
