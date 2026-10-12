package main

import "fmt"

func event(name string, n int) int { fmt.Print(name, " "); return n }

var packageValue = func() string {
	switch event("package", 2) {
	case 1:
		return "one"
	default:
		return "two"
	}
}()

func early(n int) int {
	defer fmt.Print("defer ")
	var x int
	switch n {
	case 1:
		return event("return", 7)
	default:
		x = event("value", 8)
	}
	fmt.Print("after ")
	return x
}

func main() {
	fmt.Println(packageValue)
	var x int
	switch i := event("init", 2); event("tag", i) {
	default:
		x = event("default", 99)
	case event("case1", 1), event("case2", 2):
		x = event("body", 20)
	case event("skip", 2):
		x = 30
	}
	fmt.Println(x)
	switch event("tag", 3) {
	default:
		x = event("default", 99)
	case func() int {
		if event("conditional", 1) > 0 {
			return 1
		}
		return 2
	}():
		x = 10
	case event("case2", 2):
		x = 20
	}
	fmt.Println(x)
	switch {
	case event("false", 0) > 0:
		x = 10
	case event("true", 1) > 0:
		x = 20
	default:
		x = 30
	}
	fmt.Println(x)
	for _, item := range []any{3, "four", nil} {
		switch v := item.(type) {
		case int:
			x = v + 1
		case string:
			x = len(v)
		default:
			x = 0
		}
		fmt.Println(x)
	}
	first := event("first", 1)
	var middle int
	switch 1 {
	case 1:
		middle = event("middle", 2)
	default:
		middle = 3
	}
	fmt.Println(first + middle + event("last", 3))
	fmt.Println(false && func() bool {
		switch 1 {
		case 1:
			return event("skip", 1) > 0
		default:
			return true
		}
	}())
	for i := 0; ; i++ {
		var limit int
		switch 1 {
		case 1:
			limit = event("loop", 2)
		default:
			limit = 0
		}
		if i >= limit {
			break
		}
		fmt.Print(i, " ")
	}
	fmt.Println()
	fmt.Println(early(1), early(2))
outer:
	for i := 0; i < 3; i++ {
		switch i {
		case 0:
			continue outer
		case 1:
			for j := 0; j < 1; j++ {
				break
			}
			x = 10
		default:
			break outer
		}
		fmt.Println(x)
	}
}
