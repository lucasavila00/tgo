package main

import "fmt"

func initialize(stop bool) []int {
	i := 10
	var values []func() int
	if !stop {
		for i := i; i < 12; i++ {
			values = append(values, func() int { return i })
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
	if !stop {
		for number, text := int8(1), "value"; number < 3; number++ {
			values = append(values, func() string { return fmt.Sprint(text, number) })
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
	if skip {
		i++
	}
	for ; i < 3; i++ {
		total += i
	}
	return total
}

func resumeBinding(skip bool) int {
	i := 7
	var value func() int
	start := i
	if skip {
		start = 8
	}
	for i := start; i < 9; i = 8 {
		value = func() int { return i }
		break
	}
	return value() + i
}

func restart() int {
	attempts := 0
again:
	attempts++
	if attempts == 1 {
		goto again
	}
	if attempts != 2 {
		panic("loop body")
	}
	return attempts
}

func generic[T ~int | ~int16](stop bool, value T) T {
	if !stop {
		for i := value; i < value+1; i++ {
			return i
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
