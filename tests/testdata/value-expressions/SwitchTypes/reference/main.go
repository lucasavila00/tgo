package main

import "fmt"

type small int8

func pick[T any](x, y T) T  { return x }
func identity[T any](x T) T { return x }
func exits(n int) int {
	switch n {
	case 0:
		return 10
	default:
		return 20
	}
}

func main() {
	var n small
	switch 1 {
	case 1:
		n = 127
	default:
		n = 0
	}
	fmt.Println(n, n+1)
	var p *int
	switch 1 {
	case 1:
		p = nil
	default:
		p = nil
	}
	fmt.Println(p == nil)
	var a any
	switch 1 {
	case 1:
		a = "text"
	default:
		a = 3
	}
	fmt.Println(a)
	var generic any
	switch 1 {
	case 1:
		generic = 4
	default:
		generic = "text"
	}
	fmt.Println(pick(generic, any(5)))
	var selected func(int) int
	switch 1 {
	case 1:
		selected = identity[int]
	default:
		selected = identity[int]
	}
	fmt.Println(selected(6))
	fmt.Println(exits(0), exits(1))
	var nested int
	switch 1 {
	case 1:
		switch 2 {
		case 2:
			nested = 7
		default:
			nested = 8
		}
	default:
		nested = 9
	}
	fmt.Println(nested)
	if true {
		switch 2 {
		case 2:
			nested = 8
		default:
			nested = 9
		}
	} else {
		nested = 10
	}
	fmt.Println(nested)
	xs := []int{0, 0}
	var index, left, right int
	switch 1 {
	case 1:
		index = 0
	default:
		index = 1
	}
	switch 1 {
	case 1:
		left = 11
	default:
		left = 0
	}
	switch 1 {
	case 1:
		right = 12
	default:
		right = 0
	}
	xs[index], xs[1] = left, right
	fmt.Println(xs)
	ch := make(chan int, 1)
	var send int
	switch 1 {
	case 1:
		send = 13
	default:
		send = 0
	}
	select {
	case ch <- send:
	}
	fmt.Println(<-ch)
	i := 0
	var items []int
	switch 1 {
	case 1:
		items = []int{1, 2}
	default:
		items = nil
	}
	for range items {
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
