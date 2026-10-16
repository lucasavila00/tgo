package main

import (
	"fmt"
	"testing"
)

type small struct {
	value int
	tag   uint8
}
type large struct {
	payload any
	tag     uint8
}
type data struct{ value [256]byte }

var smallSink small
var largeSink large
var result int

func makeSmall(value int) int {
	smallSink = small{value: value}
	return smallSink.value
}
func makeLarge(value [256]byte) byte {
	largeSink = large{payload: data{value: value}, tag: 1}
	return largeSink.payload.(data).value[0]
}
func main() {
	value := [256]byte{9}
	fmt.Println(makeSmall(17), makeLarge(value))
	inline := testing.AllocsPerRun(100, func() { result = makeSmall(17) })
	boxed := testing.AllocsPerRun(100, func() { result = int(makeLarge(value)) })
	fmt.Println(inline == 0, inline == 0, boxed == 1)
}
