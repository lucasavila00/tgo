package main

import (
	"fmt"
	"math"
	"unsafe"
)

type Number struct {
	tgoTag      NumberTag
	tgoVariant0 NumberInline
	tgoPayload  interface {
	}
}
type NumberTag uint8

const (
	NumberTagInline NumberTag = 0
	NumberTagBoxed  NumberTag = 1
)

func (tgo19 Number) Tag() NumberTag {
	return tgo19.tgoTag
}

type NumberInline struct {
	Float   float64
	Complex complex128
}

func NewNumberInline(tgo20 float64, tgo21 complex128) Number {
	return Number{tgoTag: NumberTagInline, tgoVariant0: NumberInline{tgo20, tgo21}}
}
func (tgo22 Number) InlinePayload() NumberInline {
	return tgo22.tgoVariant0
}

type NumberBoxed struct {
	Float   float64
	Complex complex128
	Padding [128]byte
}

func NewNumberBoxed(tgo23 float64, tgo24 complex128, tgo25 [128]byte) Number {
	return Number{tgoTag: NumberTagBoxed, tgoPayload: NumberBoxed{tgo23, tgo24, tgo25}}
}
func (tgo26 Number) BoxedPayload() NumberBoxed {
	return tgo26.tgoPayload.(NumberBoxed)
}

type LargeFirst struct {
	tgoTag      LargeFirstTag
	tgoVariant0 LargeFirstData
}
type LargeFirstTag uint8

const (
	LargeFirstTagData  LargeFirstTag = 0
	LargeFirstTagEmpty LargeFirstTag = 1
)

func (tgo27 LargeFirst) Tag() LargeFirstTag {
	return tgo27.tgoTag
}

type LargeFirstData struct{ Bytes [128]byte }

func NewLargeFirstData(tgo28 [128]byte) LargeFirst {
	return LargeFirst{tgoTag: LargeFirstTagData, tgoVariant0: LargeFirstData{tgo28}}
}
func (tgo29 LargeFirst) DataPayload() LargeFirstData {
	return tgo29.tgoVariant0
}

type LargeFirstEmpty struct{}

func NewLargeFirstEmpty() LargeFirst {
	return LargeFirst{tgoTag: LargeFirstTagEmpty}
}
func (tgo30 LargeFirst) EmptyPayload() LargeFirstEmpty {
	return LargeFirstEmpty{}
}

func signs(value float64, pair complex128) {
	fmt.Println(math.Signbit(value), math.Signbit(real(pair)), math.Signbit(imag(pair)))
}

func main() {
	var zero Number
	var payload_tgo1 NumberInline = NumberInline{}
	var enum_tgo2 Number = NewNumberInline(payload_tgo1.Float, payload_tgo1.Complex)
	first := enum_tgo2
	fmt.Println(zero == first, map[Number]string{first: "zero"}[zero])
	negative := math.Copysign(0, -1)
	pair := complex(negative, negative)
	var payload_tgo3 NumberInline = NumberInline{Float: negative, Complex: pair}
	var enum_tgo4 Number = NewNumberInline(payload_tgo3.Float, payload_tgo3.Complex)
	inline := enum_tgo4
	signs(inline.InlinePayload().Float, inline.InlinePayload().Complex)
	fmt.Println(inline == zero, map[Number]string{inline: "inline"}[zero])
	var payload_tgo5 NumberBoxed = NumberBoxed{Float: negative, Complex: pair}
	var enum_tgo6 Number = NewNumberBoxed(payload_tgo5.Float, payload_tgo5.Complex, payload_tgo5.Padding)
	boxed := enum_tgo6
	var payload_tgo7 NumberBoxed = NumberBoxed{Float: negative, Complex: pair}
	var enum_tgo8 Number = NewNumberBoxed(payload_tgo7.Float, payload_tgo7.Complex, payload_tgo7.Padding)
	other := enum_tgo8
	signs(boxed.BoxedPayload().Float, boxed.BoxedPayload().Complex)
	fmt.Println(boxed == other, map[Number]string{boxed: "boxed"}[other])
	var value_tgo11 Number = boxed
	var payload_tgo9 NumberBoxed = NumberBoxed{}
	var enum_tgo10 Number = NewNumberBoxed(payload_tgo9.Float, payload_tgo9.Complex, payload_tgo9.Padding)
	var value_tgo15 bool = value_tgo11 == enum_tgo10
	var value_tgo14 map[Number]string = map[Number]string{boxed: "positive"}
	var payload_tgo12 NumberBoxed = NumberBoxed{}
	var enum_tgo13 Number = NewNumberBoxed(payload_tgo12.Float, payload_tgo12.Complex, payload_tgo12.Padding)
	fmt.Println(value_tgo15 == (0 == 0), value_tgo14[enum_tgo13])
	var large LargeFirst
	var value_tgo18 LargeFirst = large
	var payload_tgo16 LargeFirstData = LargeFirstData{}
	var enum_tgo17 LargeFirst = NewLargeFirstData(payload_tgo16.Bytes)
	fmt.Println(unsafe.Sizeof(large) > 80, value_tgo18 == enum_tgo17)
}
