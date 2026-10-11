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
	if ok {
		return 1
	}
	return 2
}

func main() {
	point := 7
	for _, ok := range []bool{false, true} {
		var fn func(int) int
		if ok {
			fn = identity
		} else {
			fn = identity
		}
		if apply(1, fn) != 6 {
			panic("function parameter context")
		}
		first, second := pair((*int)(nil), &point)
		if first != nil || second != &point {
			panic("pointer inference")
		}
		calls := 0
		var source any
		if ok {
			calls++
			if !ok {
				source = 1
			} else {
				source = "two"
			}
		} else {
			calls++
			source = "three"
		}
		left, right := pair(source, any(4))
		var want any = "three"
		if ok {
			want = "two"
		}
		if left != want || right != 4 || calls != 1 {
			panic("interface parameter")
		}
		var number uint64
		if ok {
			number = 1
		} else {
			number = 2
		}
		count, exact := pair(number, uint64(3))
		if exact != 3 || count != 1 && ok || count != 2 && !ok {
			panic("constant conversion")
		}
		if last(&point, (*int)(nil)) != nil {
			panic("variadic inference")
		}
		var label any = 2
		var byteValue uint8 = 5
		if ok {
			label, byteValue = "one", 4
		}
		value := partial[any](label, uint8(3), any(nil), byteValue)
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
