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
	var result uintptr
	{
		if 0 != 0 {
			var tgo1 uint64
			if ok {
				goto done
			} else {
				fmt.Print("unexpected")
				tgo1 = onlyHere
			}
			_ = tgo1
		}
		result = unsafe.Sizeof(*func() (tgo2 *uint64) {
			return
		}(),
		)
	}
done:
	return result
}

func sizes[T any](value T) (uintptr, uintptr) {
	if 0 != 0 {
		var tgo3 T
		if true {
			tgo3 = value
		} else {
			tgo3 = value
		}
		_ = tgo3
	}
	var value_tgo7 uintptr = unsafe.Sizeof(*func() (tgo4 *T) {
		return
	}())
	if 0 != 0 {
		var tgo5 T
		if true {
			tgo5 = value
		} else {
			tgo5 = value
		}
		_ = tgo5
	}
	return value_tgo7, unsafe.Alignof(*func() (tgo6 *T) {
		return
	}())
}

func main() {
	if size(true) != unsafe.Sizeof(uint64(0)) || size(false) != unsafe.Sizeof(uint64(0)) {
		panic("operand ran")
	}
	size, align := sizes(layout{})
	if size != unsafe.Sizeof(layout{}) || align != unsafe.Alignof(layout{}) {
		panic("generic operand type changed")
	}
	if 0 != 0 {
		var tgo8 layout
		if true {
			fmt.Print("unexpected")
			tgo8 = layout{}
		} else {
			tgo8 = layout{}
		}
		_ = (tgo8).Next
	}
	offset := unsafe.Offsetof((*func() (tgo9 *layout) {
		return
	}()).
		Next)
	if offset != unsafe.Offsetof(layout{}.Next) {
		panic("field offset changed")
	}
	count := 0
	var tgo10 [2]int
	if true {
		count++
		tgo10 = [2]int{}
	} else {
		tgo10 = [2]int{}
	}
	length := len(tgo10)
	var tgo11 [2]int
	if true {
		count++
		tgo11 = [2]int{}
	} else {
		tgo11 = [2]int{}
	}
	for range tgo11 {
		count++
	}
	fmt.Println("ok", length, count)
}
