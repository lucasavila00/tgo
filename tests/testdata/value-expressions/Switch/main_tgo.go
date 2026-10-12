package main

import "fmt"

func event(name string, n int) int { fmt.Print(name, " "); return n }

var packageValue = func() string {
	var tgo1 string
	switch event("package", 2) {
	case 1:
		tgo1 = "one"
	default:
		tgo1 = "two"
	}
	return tgo1
}()

func early(n int) int {
	defer fmt.Print("defer ")
	var tgo2 int
	switch n {
	case 1:
		return event("return", 7)
	default:
		tgo2 = event("value", 8)
	}
	x := tgo2

	fmt.Print("after ")
	return x
}

func main() {
	fmt.Println(packageValue)
	var tgo3 int
	switch i := event("init", 2); event("tag", i) {
	default:
		tgo3 = event("default", 99)
	case event("case1", 1), event("case2", 2):
		tgo3 = event("body", 20)
	case event("skip", 2):
		tgo3 = 30
	}
	x := tgo3

	fmt.Println(x)
	var tgo4 int
	{
		var value_tgo5 int = event("tag", 3)
		var tgo6 int = -1
		if tgo6 == -1 {
			var tgo7 int

			if event("conditional", 1) > 0 {
				tgo7 = 1
			} else {
				tgo7 = 2
			}
			if value_tgo5 == tgo7 {
				tgo6 = 1
			}
		}
		if tgo6 == -1 {
			if value_tgo5 ==
				event("case2", 2) {
				tgo6 = 2
			}
		}
		if tgo6 == -1 {
			tgo6 = 0
		}
		switch tgo6 {
		case 0:
			tgo4 = event("default", 99)
		case 1:
			tgo4 = 10
		case 2:
			tgo4 = 20
		}
	}
	x = tgo4

	fmt.Println(x)
	var tgo8 int
	switch {
	case event("false", 0) > 0:
		tgo8 = 10
	case event("true", 1) > 0:
		tgo8 = 20
	default:
		tgo8 = 30
	}
	x = tgo8

	fmt.Println(x)
	for _, item := range []any{3, "four", nil} {
		var tgo9 int
		switch v := item.(type) {
		case int:
			tgo9 = v + 1
		case string:
			tgo9 = len(v)
		default:
			tgo9 = 0
		}
		x = tgo9

		fmt.Println(x)
	}
	var value_tgo11 int = event("first", 1)
	var tgo10 int
	switch 1 {
	case 1:
		tgo10 = event("middle", 2)
	default:
		tgo10 = 3
	}
	fmt.Println(value_tgo11 + (tgo10) + event("last", 3))
	var tgo13 bool
	tgo13 = false
	if tgo13 {
		var tgo12 bool
		switch 1 {
		case 1:
			tgo12 = event("skip", 1) > 0
		default:
			tgo12 = true
		}
		tgo13 = tgo12
	}
	fmt.Println(tgo13)
	{
		var tgo16 bool = 0 == 0
		for i := 0; ; {
			if tgo16 {
				tgo16 = 0 != 0
			} else {
				i++
			}
			var value_tgo15 int = i
			var tgo14 int
			switch 1 {
			case 1:
				tgo14 = event("loop", 2)
			default:
				tgo14 = 0
			}
			if !(value_tgo15 < tgo14) {
				break
			}
			{
				fmt.Print(i, " ")
			}
		}
	}
	fmt.Println()
	fmt.Println(early(1), early(2))
outer:
	for i := 0; i < 3; i++ {
		var tgo17 int
		switch i {
		case 0:
			continue outer
		case 1:
			for j := 0; j < 1; j++ {
				break
			}
			tgo17 = 10
		default:
			break outer
		}
		x = tgo17

		fmt.Println(x)
	}
}
