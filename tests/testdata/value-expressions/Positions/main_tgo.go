package main

import "fmt"

type record struct {
	value any
}

func identity[T any](value T) T {
	return value
}

func main() {
	ok := true
	var tgo1 any
	if ok {
		tgo1 = "text"
	} else {
		tgo1 = 1
	}
	var text any = tgo1
	var tgo2 any
	if ok {
		tgo2 = 1
	} else {
		tgo2 = "two"
	}
	keyed := record{value: tgo2}
	var tgo3 any
	if ok {
		tgo3 = "three"
	} else {
		tgo3 = 4
	}
	positional := record{tgo3}
	var tgo4 string
	if ok {
		tgo4 = "key"
	} else {
		tgo4 = "other"
	}
	var tgo5 any
	if ok {
		tgo5 = 5
	} else {
		tgo5 = "six"
	}
	mapping := map[string]any{tgo4: tgo5}
	var tgo6 any
	if ok {
		tgo6 = 7
	} else {
		tgo6 = "eight"
	}
	values := []any{tgo6}
	var tgo7 int
	if ok {
		tgo7 = 9
	} else {
		tgo7 = 10
	}
	array := [2]int{tgo7, 11}
	var tgo8 *record
	if ok {
		tgo8 = &record{12}
	} else {
		tgo8 = &record{13}
	}
	pointer := tgo8
	fmt.Println(text, keyed.value, positional.value, mapping["key"], values[0], array, pointer.value)

	numbers := []int{10, 20, 30}
	{
		var tgo9 []int = numbers
		var tgo10 int
		if ok {
			tgo10 = 1
		} else {
			tgo10 = 0
		}
		var tgo11 int
		if ok {
			tgo11 = 21
		} else {
			tgo11 = 22
		}
		tgo9[tgo10] = tgo11
	}
	{
		var tgo12 []int = numbers
		var tgo13 int
		if ok {
			tgo13 = 2
		} else {
			tgo13 = 0
		}
		tgo12[tgo13]++
	}
	{
		var tgo14 int
		if ok {
			tgo14 = 3
		} else {
			tgo14 = 4
		}
		var tgo17 int = -(tgo14)
		var tgo16 []int = numbers
		var tgo15 int
		if ok {
			tgo15 = 1
		} else {
			tgo15 = 0
		}
		var tgo21 int = tgo16[tgo15]
		var tgo19 []int = numbers
		var tgo18 int
		if ok {
			tgo18 = 1
		} else {
			tgo18 = 0
		}
		var tgo20 int
		if ok {
			tgo20 = 3
		} else {
			tgo20 = 2
		}
		fmt.Println(tgo17, tgo21, tgo19[tgo18:tgo20])
	}
	{
		var tgo22 int
		if ok {
			tgo22 = 14
		} else {
			tgo22 = 15
		}
		var tgo24 int = identity(tgo22)
		var tgo23 int8
		if ok {
			tgo23 = 16
		} else {
			tgo23 = 17
		}
		fmt.Println(tgo24, int8(tgo23))
	}
	var tgo25 func(value int) int
	if ok {
		tgo25 = func(value int) int { return value + 1 }
	} else {
		tgo25 = func(value int) int { return value + 2 }
	}
	call := tgo25
	{
		var tgo27 func(value int) int = call
		var tgo26 int
		if ok {
			tgo26 = 18
		} else {
			tgo26 = 19
		}
		var tgo29 int = tgo27(tgo26)
		var tgo28 any
		if ok {
			tgo28 = text
		} else {
			tgo28 = any(1)
		}
		fmt.Println(tgo29, (tgo28).(string))
	}
	{
		var tgo30 int

		if ok {
			tgo30 = 20
		} else {
			tgo30 = 0
		}
		value := tgo30
		var tgo31 bool
		if ok {
			tgo31 = value > 0
		} else {
			tgo31 = false
		}
		if tgo31 {
			fmt.Println("if", value)
		}
	}
	{
		var tgo34 int
		if ok {
			tgo34 = 0
		} else {
			tgo34 = 2
		}
		var tgo35 bool = 0 == 0
		for i := tgo34; ; {
			if tgo35 {
				tgo35 = 0 != 0
			} else {
				var tgo33 int
				if ok {
					tgo33 = 1
				} else {
					tgo33 = 2
				}
				i += tgo33
			}
			var tgo32 bool
			if ok {
				tgo32 = i < 2
			} else {
				tgo32 = false
			}
			if !tgo32 {
				break
			}
			{
				fmt.Println("for", i)
			}
		}
	}
	{
		var tgo36 []int
		if ok {
			tgo36 = []int{22, 23}
		} else {
			tgo36 = []int{}
		}
		for _, value := range tgo36 {
			fmt.Println("range", value)
		}
	}
	{
		var tgo37 int
		if ok {
			tgo37 = 24
		} else {
			tgo37 = 0
		}
		value := tgo37
		var tgo38 int
		if ok {
			tgo38 = value
		} else {
			tgo38 = 0
		}
		var tgo39 int = tgo38
		var tgo40 int = -1
		if tgo40 == -1 {
			var tgo41 int
			if ok {
				tgo41 = 24
			} else {
				tgo41 = 25
			}
			if tgo39 == tgo41 {
				tgo40 = 0
			}
		}
		if tgo40 == -1 {
			tgo40 = 1
		}
		switch tgo40 {
		case 0:
			fmt.Println("switch", value)
		case 1:
			panic("wrong case")
		}
	}
	{
		var tgo42 any

		if ok {
			tgo42 = text
		} else {
			tgo42 = any(1)
		}
		switch value := (tgo42).(type) {
		case string:
			fmt.Println("type", value)
		default:
			panic("wrong type")
		}
	}

	channel := make(chan any, 1)
	{
		var tgo44 chan any = channel
		var tgo43 any
		if ok {
			tgo43 = 26
		} else {
			tgo43 = "other"
		}
		tgo44 <- tgo43
	}
	fmt.Println("send", <-channel)
	{
		var tgo45 chan any = channel
		var tgo46 any
		if ok {
			tgo46 = 27
		} else {
			tgo46 = "other"
		}
		select {
		case tgo45 <- tgo46:
			fmt.Println("select", <-channel)
		default:
			panic("blocked send")
		}
	}
	{
		var tgo48 func(value any) = func(value any) { channel <- value }
		var tgo47 any
		if ok {
			tgo47 = 28
		} else {
			tgo47 = "other"
		}
		go tgo48(tgo47)
	}
	fmt.Println("go", <-channel)
	{
		var tgo49 any
		if ok {
			tgo49 = 29
		} else {
			tgo49 = "other"
		}
		defer fmt.Println("defer", tgo49)
	}
}
