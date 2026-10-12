package main

import "fmt"

func initialize(stop bool) []int {
	i := 10
	var values []func() int
	{
		var tgo3 int
		var tgo4 bool = 0 == 0
	loop:
		for i_tgo2 := tgo3; ; {
			if tgo4 {
				tgo4 = 0 != 0
				var tgo1 int
				if stop {
					break loop
				} else {
					tgo1 = func() int { return i }()
				}
				i_tgo2 = tgo1
			} else {

				i_tgo2++
			}
			if !(i_tgo2 < 12) {
				break
			}
			{
				values = append(values, func() int { return i_tgo2 })
				{
					i := 50
					if i != 50 {
						panic("shadowed binding")
					}
				}
			}
		}
	}
	result := []int{i}
	for _, value := range values {
		result = append(result, value())
	}
	return result
}

func initializeMany(stop bool) []string {
	var values []func() string
	{
		var tgo8 int8
		var tgo10 string
		var tgo11 bool = 0 == 0
		for number_tgo7, text_tgo9 := tgo8, tgo10; ; {
			if tgo11 {
				tgo11 = 0 != 0
				var tgo5 int8
				if stop {
					break
				} else {
					tgo5 = int8(1)
				}
				var tgo6 string
				if stop {
					panic("second operand")
				} else {
					tgo6 = "value"
				}
				number_tgo7, text_tgo9 = tgo5, tgo6
			} else {

				number_tgo7++
			}
			if !(number_tgo7 < 3) {
				break
			}
			{
				values = append(values, func() string { return fmt.Sprint(text_tgo9, number_tgo7) })
			}
		}
	}
	var result []string
	for _, value := range values {
		result = append(result, value())
	}
	return result
}

func resume(skip bool) int {
	i, total := 0, 0
	{
		var tgo13 bool = 0 == 0
	loop:
		for {
			if tgo13 {
				tgo13 = 0 != 0
				var tgo12 int
				if skip {
					continue loop
				} else {
					tgo12 = 0
				}
				_ = tgo12
			} else {

				i++
			}
			if !(i < 3) {
				break
			}
			{
				total += i
			}
		}
	}
	return total
}

func resumeBinding(skip bool) int {
	i := 7
	var value func() int
	{
		var tgo16 int
		var tgo17 bool = 0 == 0
		for i_tgo15 := tgo16; ; {
			if tgo17 {
				tgo17 = 0 != 0
				var tgo14 int
				if skip {
					continue
				} else {
					tgo14 = i
				}
				i_tgo15 = tgo14
			} else {

				i_tgo15 = 8
			}
			if !(i_tgo15 < 9) {
				break
			}
			{
				value = func() int { return i_tgo15 }
				break
			}
		}
	}
	return value() + i
}

func restart() int {
	attempts := 0
again:
	{
		var tgo21 int
		var tgo22 bool = 0 == 0
	tgo18:
		for i_tgo20 := tgo21; ; {
			if tgo22 {
				tgo22 = 0 != 0
				var tgo19 int
				if attempts++; attempts == 1 {
					goto again
				} else if attempts == 2 {
					break tgo18
				} else {
					tgo19 = 0
				}
				i_tgo20 = tgo19
			} else {

				i_tgo20++
			}
			if !(i_tgo20 < 1) {
				break
			}
			{
				panic("loop body")
			}
		}
	}
	return attempts
}

func generic[T ~int | ~int16](stop bool, value T) T {
	{
		var tgo25 T
		var tgo26 bool = 0 == 0
	loop:
		for i_tgo24 := tgo25; ; {
			if tgo26 {
				tgo26 = 0 != 0
				var tgo23 T
				if stop {
					break loop
				} else {
					tgo23 = value
				}
				i_tgo24 = tgo23
			} else {

				i_tgo24++
			}
			if !(i_tgo24 < value+1) {
				break
			}
			{
				return i_tgo24
			}
		}
	}
	return value
}

func main() {
	for _, stop := range []bool{false, true} {
		values := initialize(stop)
		many := initializeMany(stop)
		if stop {
			if len(many) != 0 {
				panic("multiple bindings break")
			}
		} else if len(many) != 2 || many[0] != "value1" || many[1] != "value2" {
			panic("multiple iteration bindings")
		}
		if values[0] != 10 {
			panic("outer binding")
		}
		if stop {
			if len(values) != 1 {
				panic("initializer break")
			}
		} else if len(values) != 3 || values[1] != 10 || values[2] != 11 {
			panic("iteration bindings")
		}
		if resume(stop) != 3 {
			panic("initializer continue")
		}
		want := 14
		if stop {
			want = 15
		}
		if resumeBinding(stop) != want {
			panic("continued initializer binding")
		}
		if generic(stop, int16(9)) != 9 {
			panic("generic initializer")
		}
	}
	if restart() != 2 {
		panic("initializer goto")
	}
	fmt.Println("ok")
}
