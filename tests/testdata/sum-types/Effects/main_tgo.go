package main

import (
	"errors"
	"fmt"
)

var trace []string
var failure = errors.New("failed")

type Pair struct {
	tgoTag      PairTag
	tgoVariant0 PairValues
}
type PairTag uint8

const (
	PairTagValues PairTag = 0
	PairTagEmpty  PairTag = 1
)

func (tgo18 Pair) Tag() PairTag {
	return tgo18.tgoTag
}

type PairValues struct {
	Left  int
	Right int
}

func NewPairValues(tgo19 int, tgo20 int) Pair {
	return Pair{tgoTag: PairTagValues, tgoVariant0: PairValues{tgo19, tgo20}}
}
func (tgo21 Pair) ValuesPayload() PairValues {
	return tgo21.tgoVariant0
}

type PairEmpty struct{}

func NewPairEmpty() Pair {
	return Pair{tgoTag: PairTagEmpty}
}
func (tgo22 Pair) EmptyPayload() PairEmpty {
	return PairEmpty{}
}

func mark(name string, value int) int { trace = append(trace, name); return value }
func read(fail bool) (int, error) {
	trace = append(trace, "read")
	if fail {
		return 0, failure
	}
	return 4, nil
}
func pick(value Pair) Pair { trace = append(trace, "pick"); return value }
func build(fail bool) (Pair, error) {
	var zero_tgo3 Pair
	var value_tgo4 int = mark("right", 2)
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = read(fail)
	if error_tgo2 != nil {
		return zero_tgo3, error_tgo2
	}
	var payload_tgo5 PairValues = PairValues{Right: value_tgo4, Left: value_tgo1}
	var enum_tgo6 Pair = NewPairValues(payload_tgo5.Left, payload_tgo5.Right)
	return enum_tgo6, nil
}
func main() {
	var payload_tgo7 PairValues = PairValues{Right: mark("right", 2), Left: mark("left", 1)}
	var enum_tgo8 Pair = NewPairValues(payload_tgo7.Left, payload_tgo7.Right)
	value := enum_tgo8
	fmt.Println(value.ValuesPayload(), trace)
	trace = nil
	var payload_tgo9 PairValues = PairValues{Right: mark("address-right", 3), Left: mark("address-left", 2)}
	var enum_tgo10 Pair = NewPairValues(payload_tgo9.Left, payload_tgo9.Right)
	var value_tgo11 *Pair = &enum_tgo10
	pointer := value_tgo11
	fmt.Println(pointer.ValuesPayload(), trace)
	trace = nil
	var tgo12 int

	if true {
		tgo12 = mark("if", 5)
	} else {
		tgo12 = 0
	}
	var tgo13 int
	switch 1 {
	case 1:
		tgo13 = mark("switch", 6)
	default:
		tgo13 = 0
	}
	var payload_tgo14 PairValues = PairValues{
		Left: tgo12,

		Right: tgo13,
	}
	var enum_tgo15 Pair = NewPairValues(payload_tgo14.Left, payload_tgo14.Right)
	value = enum_tgo15

	fmt.Println(value.ValuesPayload(), trace)
	trace = nil
	{
		var value_tgo16 Pair = pick(value)
		switch value_tgo16.Tag() {
		case PairTagValues:
			var v_tgo17 PairValues = value_tgo16.ValuesPayload()
			fmt.Println(v_tgo17.Left, v_tgo17.Right, trace)
		default:
			panic("empty")
		}
	}

	trace = nil
	good, err := build(false)
	fmt.Println(good.ValuesPayload(), err, trace)
	trace = nil
	_, err = build(true)
	fmt.Println(err == failure, trace)
}
