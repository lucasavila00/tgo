package main

import (
	"fmt"
	"math"
	"unsafe"
)

type number struct {
	boxed   bool
	value   float64
	pair    complex128
	padding [128]byte
}

type largeFirst struct {
	bytes [128]byte
	empty bool
}

func signs(value float64, pair complex128) {
	fmt.Println(math.Signbit(value), math.Signbit(real(pair)), math.Signbit(imag(pair)))
}

func main() {
	var zero number
	first := number{}
	fmt.Println(zero == first, map[number]string{first: "zero"}[zero])
	negative := math.Copysign(0, -1)
	pair := complex(negative, negative)
	inline := number{value: negative, pair: pair}
	signs(inline.value, inline.pair)
	fmt.Println(inline == zero, map[number]string{inline: "inline"}[zero])
	boxed := number{boxed: true, value: negative, pair: pair}
	other := number{boxed: true, value: negative, pair: pair}
	signs(boxed.value, boxed.pair)
	fmt.Println(boxed == other, map[number]string{boxed: "boxed"}[other])
	fmt.Println(boxed == number{boxed: true}, map[number]string{boxed: "positive"}[number{boxed: true}])
	var large largeFirst
	fmt.Println(unsafe.Sizeof(large) > 80, large == largeFirst{})
}
