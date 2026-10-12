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
	var tgo1 interface {
	}
	if ok {
		tgo1 = 30
	} else {
		tgo1 = "other"
	}
	panic(tgo1)
}

func remove[M ~map[*int]int](mapping M, ok bool) {
	var value_tgo3 M = mapping
	var tgo2 *int
	if ok {
		tgo2 = nil
	} else {
		tgo2 = nil
	}
	delete(value_tgo3, tgo2)
}

func checkCopy(ok bool) {
	type integers []int
	dst := integers{1, 2}
	calls := 0
	var value_tgo5 integers = dst
	var tgo4 []int
	if ok {
		calls++
		tgo4 = nil
	} else {
		calls++
		tgo4 = nil
	}
	n := copy(value_tgo5, tgo4)
	var value_tgo7 integers = dst
	var tgo6 []int
	if ok {
		calls++
		tgo6 = nil
	} else {
		calls++
		tgo6 = nil
	}
	values := append(value_tgo7, (tgo6)...)
	bytes := []byte{0, 0, 0}
	var value_tgo9 []byte = bytes
	var tgo8 string
	if ok {
		tgo8 = "one"
	} else {
		tgo8 = "two"
	}
	copied := copy(value_tgo9, tgo8)
	var value_tgo11 []byte = bytes
	var tgo10 []byte
	if ok {
		tgo10 = nil
	} else {
		tgo10 = nil
	}
	unchanged := append(value_tgo11, (tgo10)...)
	{
		var tgo14 bool
		tgo14 = copied != 3 || len(unchanged) != 3
		if !tgo14 {
			var value_tgo13 []byte = bytes
			var tgo12 []byte
			if ok {
				tgo12 = nil
			} else {
				tgo12 = nil
			}
			tgo14 = copy(value_tgo13, tgo12) != 0
		}
		if tgo14 == (0 == 0) {
			panic("wrong byte operation")
		}
	}
	if calls != 2 || n != 0 || len(values) != 2 || values[0] != 1 || values[1] != 2 {
		panic("wrong slice operation")
	}
}

func main() {
	checkCopy(true)
	checkCopy(false)
	checkPanic(true)
	checkPanic(false)
	mappingWithNil := map[*int]int{nil: 30}
	remove(mappingWithNil, true)
	mappingWithNil[nil] = 31
	remove(mappingWithNil, false)
	fmt.Println("delete", len(mappingWithNil))
	ok := true
	var tgo15 any
	if ok {
		tgo15 = "text"
	} else {
		tgo15 = 1
	}
	var text any = tgo15
	var tgo16 any
	if ok {
		tgo16 = 1
	} else {
		tgo16 = "two"
	}
	keyed := record{value: tgo16}
	var tgo17 any
	if ok {
		tgo17 = "three"
	} else {
		tgo17 = 4
	}
	positional := record{tgo17}
	var tgo18 string
	if ok {
		tgo18 = "key"
	} else {
		tgo18 = "other"
	}
	var tgo19 any
	if ok {
		tgo19 = 5
	} else {
		tgo19 = "six"
	}
	mapping := map[string]any{tgo18: tgo19}
	var tgo20 any
	if ok {
		tgo20 = 7
	} else {
		tgo20 = "eight"
	}
	values := []any{tgo20}
	var tgo21 int
	if ok {
		tgo21 = 9
	} else {
		tgo21 = 10
	}
	array := [2]int{tgo21, 11}
	var tgo22 *record
	if ok {
		tgo22 = &record{12}
	} else {
		tgo22 = &record{13}
	}
	pointer := tgo22
	fmt.Println(text, keyed.value, positional.value, mapping["key"], values[0], array, pointer.value)

	numbers := []int{10, 20, 30}
	var value_tgo23 []int = numbers
	var tgo24 int
	if ok {
		tgo24 = 1
	} else {
		tgo24 = 0
	}
	var tgo25 int
	if ok {
		tgo25 = 21
	} else {
		tgo25 = 22
	}
	value_tgo23[tgo24] = tgo25
	var value_tgo26 []int = numbers
	var tgo27 int
	if ok {
		tgo27 = 2
	} else {
		tgo27 = 0
	}
	value_tgo26[tgo27]++
	var tgo28 int
	if ok {
		tgo28 = 3
	} else {
		tgo28 = 4
	}
	var value_tgo31 int = -(tgo28)
	var value_tgo30 []int = numbers
	var tgo29 int
	if ok {
		tgo29 = 1
	} else {
		tgo29 = 0
	}
	var value_tgo35 int = value_tgo30[tgo29]
	var value_tgo33 []int = numbers
	var tgo32 int
	if ok {
		tgo32 = 1
	} else {
		tgo32 = 0
	}
	var tgo34 int
	if ok {
		tgo34 = 3
	} else {
		tgo34 = 2
	}
	fmt.Println(value_tgo31, value_tgo35, value_tgo33[tgo32:tgo34])
	var tgo36 int
	if ok {
		tgo36 = 14
	} else {
		tgo36 = 15
	}
	var value_tgo38 int = identity(tgo36)
	var tgo37 int8
	if ok {
		tgo37 = 16
	} else {
		tgo37 = 17
	}
	fmt.Println(value_tgo38, int8(tgo37))
	var tgo39 func(value int) int
	if ok {
		tgo39 = func(value int) int { return value + 1 }
	} else {
		tgo39 = func(value int) int { return value + 2 }
	}
	call := tgo39
	var value_tgo41 func(value int) int = call
	var tgo40 int
	if ok {
		tgo40 = 18
	} else {
		tgo40 = 19
	}
	var value_tgo43 int = value_tgo41(tgo40)
	var tgo42 any
	if ok {
		tgo42 = text
	} else {
		tgo42 = any(1)
	}
	fmt.Println(value_tgo43, (tgo42).(string))
	{
		var tgo44 int

		if ok {
			tgo44 = 20
		} else {
			tgo44 = 0
		}
		value := tgo44
		var tgo45 bool
		if ok {
			tgo45 = value > 0
		} else {
			tgo45 = false
		}
		if tgo45 {
			fmt.Println("if", value)
		}
	}
	{
		var tgo48 int
		if ok {
			tgo48 = 0
		} else {
			tgo48 = 2
		}
		var tgo49 bool = 0 == 0
		for i := tgo48; ; {
			if tgo49 {
				tgo49 = 0 != 0
			} else {
				var tgo47 int
				if ok {
					tgo47 = 1
				} else {
					tgo47 = 2
				}
				i += tgo47
			}
			var tgo46 bool
			if ok {
				tgo46 = i < 2
			} else {
				tgo46 = false
			}
			if !tgo46 {
				break
			}
			{
				fmt.Println("for", i)
			}
		}
	}
	var tgo50 []int
	if ok {
		tgo50 = []int{22, 23}
	} else {
		tgo50 = []int{}
	}
	for _, value := range tgo50 {
		fmt.Println("range", value)
	}
	{
		var tgo51 int
		if ok {
			tgo51 = 24
		} else {
			tgo51 = 0
		}
		value := tgo51
		var tgo52 int
		if ok {
			tgo52 = value
		} else {
			tgo52 = 0
		}
		var value_tgo53 int = tgo52
		var tgo54 int = -1
		if tgo54 == -1 {
			var tgo55 int
			if ok {
				tgo55 = 24
			} else {
				tgo55 = 25
			}
			if value_tgo53 == tgo55 {
				tgo54 = 0
			}
		}
		if tgo54 == -1 {
			tgo54 = 1
		}
		switch tgo54 {
		case 0:
			fmt.Println("switch", value)
		case 1:
			panic("wrong case")
		}
	}
	{
		var tgo56 any

		if ok {
			tgo56 = text
		} else {
			tgo56 = any(1)
		}
		switch value := (tgo56).(type) {
		case string:
			fmt.Println("type", value)
		default:
			panic("wrong type")
		}
	}

	channel := make(chan any, 1)
	var value_tgo58 chan any = channel
	var tgo57 any
	if ok {
		tgo57 = 26
	} else {
		tgo57 = "other"
	}
	value_tgo58 <- tgo57
	fmt.Println("send", <-channel)
	var value_tgo59 chan any = channel
	var tgo60 any
	if ok {
		tgo60 = 27
	} else {
		tgo60 = "other"
	}
	select {
	case value_tgo59 <- tgo60:
		fmt.Println("select", <-channel)
	default:
		panic("blocked send")
	}
	var value_tgo62 func(value any) = func(value any) { channel <- value }
	var tgo61 any
	if ok {
		tgo61 = 28
	} else {
		tgo61 = "other"
	}
	go value_tgo62(tgo61)
	fmt.Println("go", <-channel)
	var tgo63 any
	if ok {
		tgo63 = 29
	} else {
		tgo63 = "other"
	}
	defer fmt.Println("defer", tgo63)
}
