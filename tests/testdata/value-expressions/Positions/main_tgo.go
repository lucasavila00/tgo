package main

import "fmt"

type record struct {
	value any
}

func identity[T any](value T) T {
	return value
}

func checkPanic(ok bool) {
	defer func() {
		var want any = 30
		if !ok {
			want = "other"
		}
		if recover() != want {
			panic("wrong panic value")
		}
	}()
	{
		var tgo1 interface {
		}
		if ok {
			tgo1 = 30
		} else {
			tgo1 = "other"
		}
		panic(tgo1)
	}
}

func remove[M ~map[*int]int](mapping M, ok bool) {
	{
		var tgo3 M = mapping
		var tgo2 *int
		if ok {
			tgo2 = nil
		} else {
			tgo2 = nil
		}
		delete(tgo3, tgo2)
	}
}

func main() {
	checkPanic(true)
	checkPanic(false)
	mappingWithNil := map[*int]int{nil: 30}
	remove(mappingWithNil, true)
	mappingWithNil[nil] = 31
	remove(mappingWithNil, false)
	fmt.Println("delete", len(mappingWithNil))
	ok := true
	var tgo4 any
	if ok {
		tgo4 = "text"
	} else {
		tgo4 = 1
	}
	var text any = tgo4
	var tgo5 any
	if ok {
		tgo5 = 1
	} else {
		tgo5 = "two"
	}
	keyed := record{value: tgo5}
	var tgo6 any
	if ok {
		tgo6 = "three"
	} else {
		tgo6 = 4
	}
	positional := record{tgo6}
	var tgo7 string
	if ok {
		tgo7 = "key"
	} else {
		tgo7 = "other"
	}
	var tgo8 any
	if ok {
		tgo8 = 5
	} else {
		tgo8 = "six"
	}
	mapping := map[string]any{tgo7: tgo8}
	var tgo9 any
	if ok {
		tgo9 = 7
	} else {
		tgo9 = "eight"
	}
	values := []any{tgo9}
	var tgo10 int
	if ok {
		tgo10 = 9
	} else {
		tgo10 = 10
	}
	array := [2]int{tgo10, 11}
	var tgo11 *record
	if ok {
		tgo11 = &record{12}
	} else {
		tgo11 = &record{13}
	}
	pointer := tgo11
	fmt.Println(text, keyed.value, positional.value, mapping["key"], values[0], array, pointer.value)

	numbers := []int{10, 20, 30}
	{
		var tgo12 []int = numbers
		var tgo13 int
		if ok {
			tgo13 = 1
		} else {
			tgo13 = 0
		}
		var tgo14 int
		if ok {
			tgo14 = 21
		} else {
			tgo14 = 22
		}
		tgo12[tgo13] = tgo14
	}
	{
		var tgo15 []int = numbers
		var tgo16 int
		if ok {
			tgo16 = 2
		} else {
			tgo16 = 0
		}
		tgo15[tgo16]++
	}
	{
		var tgo17 int
		if ok {
			tgo17 = 3
		} else {
			tgo17 = 4
		}
		var tgo20 int = -(tgo17)
		var tgo19 []int = numbers
		var tgo18 int
		if ok {
			tgo18 = 1
		} else {
			tgo18 = 0
		}
		var tgo24 int = tgo19[tgo18]
		var tgo22 []int = numbers
		var tgo21 int
		if ok {
			tgo21 = 1
		} else {
			tgo21 = 0
		}
		var tgo23 int
		if ok {
			tgo23 = 3
		} else {
			tgo23 = 2
		}
		fmt.Println(tgo20, tgo24, tgo22[tgo21:tgo23])
	}
	{
		var tgo25 int
		if ok {
			tgo25 = 14
		} else {
			tgo25 = 15
		}
		var tgo27 int = identity(tgo25)
		var tgo26 int8
		if ok {
			tgo26 = 16
		} else {
			tgo26 = 17
		}
		fmt.Println(tgo27, int8(tgo26))
	}
	var tgo28 func(value int) int
	if ok {
		tgo28 = func(value int) int { return value + 1 }
	} else {
		tgo28 = func(value int) int { return value + 2 }
	}
	call := tgo28
	{
		var tgo30 func(value int) int = call
		var tgo29 int
		if ok {
			tgo29 = 18
		} else {
			tgo29 = 19
		}
		var tgo32 int = tgo30(tgo29)
		var tgo31 any
		if ok {
			tgo31 = text
		} else {
			tgo31 = any(1)
		}
		fmt.Println(tgo32, (tgo31).(string))
	}
	{
		var tgo33 int

		if ok {
			tgo33 = 20
		} else {
			tgo33 = 0
		}
		value := tgo33
		var tgo34 bool
		if ok {
			tgo34 = value > 0
		} else {
			tgo34 = false
		}
		if tgo34 {
			fmt.Println("if", value)
		}
	}
	{
		var tgo37 int
		if ok {
			tgo37 = 0
		} else {
			tgo37 = 2
		}
		var tgo38 bool = 0 == 0
		for i := tgo37; ; {
			if tgo38 {
				tgo38 = 0 != 0
			} else {
				var tgo36 int
				if ok {
					tgo36 = 1
				} else {
					tgo36 = 2
				}
				i += tgo36
			}
			var tgo35 bool
			if ok {
				tgo35 = i < 2
			} else {
				tgo35 = false
			}
			if !tgo35 {
				break
			}
			{
				fmt.Println("for", i)
			}
		}
	}
	{
		var tgo39 []int
		if ok {
			tgo39 = []int{22, 23}
		} else {
			tgo39 = []int{}
		}
		for _, value := range tgo39 {
			fmt.Println("range", value)
		}
	}
	{
		var tgo40 int
		if ok {
			tgo40 = 24
		} else {
			tgo40 = 0
		}
		value := tgo40
		var tgo41 int
		if ok {
			tgo41 = value
		} else {
			tgo41 = 0
		}
		var tgo42 int = tgo41
		var tgo43 int = -1
		if tgo43 == -1 {
			var tgo44 int
			if ok {
				tgo44 = 24
			} else {
				tgo44 = 25
			}
			if tgo42 == tgo44 {
				tgo43 = 0
			}
		}
		if tgo43 == -1 {
			tgo43 = 1
		}
		switch tgo43 {
		case 0:
			fmt.Println("switch", value)
		case 1:
			panic("wrong case")
		}
	}
	{
		var tgo45 any

		if ok {
			tgo45 = text
		} else {
			tgo45 = any(1)
		}
		switch value := (tgo45).(type) {
		case string:
			fmt.Println("type", value)
		default:
			panic("wrong type")
		}
	}

	channel := make(chan any, 1)
	{
		var tgo47 chan any = channel
		var tgo46 any
		if ok {
			tgo46 = 26
		} else {
			tgo46 = "other"
		}
		tgo47 <- tgo46
	}
	fmt.Println("send", <-channel)
	{
		var tgo48 chan any = channel
		var tgo49 any
		if ok {
			tgo49 = 27
		} else {
			tgo49 = "other"
		}
		select {
		case tgo48 <- tgo49:
			fmt.Println("select", <-channel)
		default:
			panic("blocked send")
		}
	}
	{
		var tgo51 func(value any) = func(value any) { channel <- value }
		var tgo50 any
		if ok {
			tgo50 = 28
		} else {
			tgo50 = "other"
		}
		go tgo51(tgo50)
	}
	fmt.Println("go", <-channel)
	{
		var tgo52 any
		if ok {
			tgo52 = 29
		} else {
			tgo52 = "other"
		}
		defer fmt.Println("defer", tgo52)
	}
}
