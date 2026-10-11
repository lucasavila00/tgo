package main

import "fmt"

func pair[T any](first, second T) (T, T) {
	return first, second
}

func last[T any](first T, rest ...T) T {
	return rest[len(rest)-1]
}

func partial[A, B any](first A, second B, third A, fourth B) B {
	return fourth
}

func identity[T any](value T) T {
	return value
}

func apply[T any](value T, fn func(int) int) int {
	return fn(6)
}

func exits(ok bool) uint64 {
	var tgo1 uint64
	if ok {
		return 1
	} else {
		return 2
	}
	_, value := pair(tgo1, uint64(3))
	return value
}

func main() {
	point := 7
	for _, ok := range []bool{false, true} {
		{
			var tgo2 func(int) int
			if ok {
				tgo2 = identity
			} else {
				tgo2 = identity
			}
			if apply(1, tgo2) != 6 {
				panic("function parameter context")
			}
		}
		var tgo3 *int
		if ok {
			tgo3 = nil
		} else {
			tgo3 = nil
		}
		var tgo4 *int
		if ok {
			tgo4 = &point
		} else {
			tgo4 = &point
		}
		first, second := pair(tgo3, tgo4)
		if first != nil || second != &point {
			panic("pointer inference")
		}
		calls := 0
		var tgo5 any
		if ok {
			calls++
			var tgo6 any
			if !ok {
				tgo6 = 1
			} else {
				tgo6 = "two"
			}
			tgo5 = (tgo6)
		} else {
			calls++
			tgo5 = "three"
		}
		left, right := pair(tgo5, any(4))
		var want any = "three"
		if ok {
			want = "two"
		}
		if left != want || right != 4 || calls != 1 {
			panic("interface parameter")
		}
		var tgo7 uint64
		if ok {
			tgo7 = 1
		} else {
			tgo7 = 2
		}
		count, exact := pair(tgo7, uint64(3))
		if exact != 3 || count != 1 && ok || count != 2 && !ok {
			panic("constant conversion")
		}
		{
			var tgo9 *int = &point
			var tgo8 *int
			if ok {
				tgo8 = nil
			} else {
				tgo8 = nil
			}
			if last(tgo9, tgo8) != nil {
				panic("variadic inference")
			}
		}
		var tgo10 any
		if ok {
			tgo10 = "one"
		} else {
			tgo10 = 2
		}
		var tgo12 any = any(nil)
		var tgo11 uint8
		if ok {
			tgo11 = 4
		} else {
			tgo11 = 5
		}
		value := partial[any](tgo10, uint8(3), tgo12, tgo11)
		if ok && value != 4 || !ok && value != 5 {
			panic("partial type arguments")
		}
		wantExit := uint64(2)
		if ok {
			wantExit = 1
		}
		if exits(ok) != wantExit {
			panic("terminating argument")
		}
	}
	fmt.Println("ok")
}
