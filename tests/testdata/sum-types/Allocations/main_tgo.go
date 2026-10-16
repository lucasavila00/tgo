package main

import (
	"fmt"
	"testing"
)

type Small struct {
	tgoTag      SmallTag
	tgoVariant0 SmallNumber
}
type SmallTag uint8

const (
	SmallTagNumber SmallTag = 0
	SmallTagEmpty  SmallTag = 1
)

func (tgo5 Small) Tag() SmallTag {
	return tgo5.tgoTag
}

type SmallNumber struct{ Value int }

func NewSmallNumber(tgo6 int) Small {
	return Small{tgoTag: SmallTagNumber, tgoVariant0: SmallNumber{tgo6}}
}
func (tgo7 Small) NumberPayload() SmallNumber {
	return tgo7.tgoVariant0
}

type SmallEmpty struct{}

func NewSmallEmpty() Small {
	return Small{tgoTag: SmallTagEmpty}
}
func (tgo8 Small) EmptyPayload() SmallEmpty {
	return SmallEmpty{}
}

type Large struct {
	tgoTag     LargeTag
	tgoPayload interface {
	}
}
type LargeTag uint8

const (
	LargeTagEmpty LargeTag = 0
	LargeTagData  LargeTag = 1
)

func (tgo9 Large) Tag() LargeTag {
	return tgo9.tgoTag
}

type LargeEmpty struct{}

func NewLargeEmpty() Large {
	return Large{tgoTag: LargeTagEmpty}
}
func (tgo10 Large) EmptyPayload() LargeEmpty {
	return LargeEmpty{}
}

type LargeData struct{ Value [256]byte }

func NewLargeData(tgo11 [256]byte) Large {
	return Large{tgoTag: LargeTagData, tgoPayload: LargeData{tgo11}}
}
func (tgo12 Large) DataPayload() LargeData {
	return tgo12.tgoPayload.(LargeData)
}

type smallGo struct {
	value int
	tag   uint8
}
type largeGo struct {
	payload any
	tag     uint8
}
type dataGo struct{ value [256]byte }

var smallSink Small
var largeSink Large
var smallGoSink smallGo
var largeGoSink largeGo
var result int

func makeSmallTGo(value int) int {
	var payload_tgo1 SmallNumber = SmallNumber{Value: value}
	var enum_tgo2 Small = NewSmallNumber(payload_tgo1.Value)
	smallSink = enum_tgo2
	return smallSink.NumberPayload().Value
}
func makeSmallGo(value int) int {
	smallGoSink = smallGo{value: value}
	return smallGoSink.value
}
func makeLargeTGo(value [256]byte) byte {
	var payload_tgo3 LargeData = LargeData{Value: value}
	var enum_tgo4 Large = NewLargeData(payload_tgo3.Value)
	largeSink = enum_tgo4
	return largeSink.DataPayload().Value[0]
}
func makeLargeGo(value [256]byte) byte {
	largeGoSink = largeGo{payload: dataGo{value: value}, tag: 1}
	return largeGoSink.payload.(dataGo).value[0]
}
func main() {
	value := [256]byte{9}
	fmt.Println(makeSmallTGo(17), makeLargeTGo(value))
	inline := testing.AllocsPerRun(100, func() { result = makeSmallTGo(17) })
	inlineGo := testing.AllocsPerRun(100, func() { result = makeSmallGo(17) })
	boxed := testing.AllocsPerRun(100, func() { result = int(makeLargeTGo(value)) })
	boxedGo := testing.AllocsPerRun(100, func() { result = int(makeLargeGo(value)) })
	fmt.Println(inline == 0, inline == inlineGo, boxed == boxedGo)
}
