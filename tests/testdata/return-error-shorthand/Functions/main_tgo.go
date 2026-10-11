package main

import (
	"fmt"
	"unsafe"
)

var callback = func() (int, error) { return 1, nil }
var value, initialErr = func() (int, error) { return 2, nil }()
var callbacks = []func() (int, error){
	func() (int, error) { return 3, nil },
}

const size = unsafe.Sizeof(func() (int, error) { return 4, nil })
const length = len([1]func() (int, error){func() (int, error) { return 5, nil }})

func main() {
	number, err := callback()
	fmt.Println(number, err)
	fmt.Println(value, initialErr)
	fmt.Println(size > 0, length)
	for _, fn := range callbacks {
		number, err = fn()
		fmt.Println(number, err)
	}
}
