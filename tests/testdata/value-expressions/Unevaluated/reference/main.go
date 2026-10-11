package main

import (
	"fmt"
	"unsafe"
)

type layout struct {
	First byte
	Next  uint64
}

func size(ok bool) uintptr {
	onlyHere := uint64(42)
	return unsafe.Sizeof(onlyHere)
}

func sizes[T any](value T) (uintptr, uintptr) {
	return unsafe.Sizeof(value), unsafe.Alignof(value)
}

func main() {
	if size(true) != unsafe.Sizeof(uint64(0)) || size(false) != unsafe.Sizeof(uint64(0)) {
		panic("operand ran")
	}
	size, align := sizes(layout{})
	if size != unsafe.Sizeof(layout{}) || align != unsafe.Alignof(layout{}) {
		panic("generic operand type changed")
	}
	offset := unsafe.Offsetof(layout{}.Next)
	if offset != unsafe.Offsetof(layout{}.Next) {
		panic("field offset changed")
	}
	count := 0
	var value [2]int
	if true {
		count++
		value = [2]int{}
	} else {
		value = [2]int{}
	}
	length := len(value)
	if true {
		count++
		value = [2]int{}
	} else {
		value = [2]int{}
	}
	for range value {
		count++
	}
	fmt.Println("ok", length, count)
}
