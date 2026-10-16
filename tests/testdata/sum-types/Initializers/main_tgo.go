package main

import "fmt"

type Choice struct {
	tgoTag      ChoiceTag
	tgoVariant0 ChoiceNumber
	tgoVariant1 ChoiceText
}
type ChoiceTag uint8

const (
	ChoiceTagNumber ChoiceTag = 0
	ChoiceTagText   ChoiceTag = 1
)

func (tgo12 Choice) Tag() ChoiceTag {
	return tgo12.tgoTag
}

type ChoiceNumber struct{ Value int }

func NewChoiceNumber(tgo13 int) Choice {
	return Choice{tgoTag: ChoiceTagNumber, tgoVariant0: ChoiceNumber{tgo13}}
}
func (tgo14 Choice) NumberPayload() ChoiceNumber {
	return tgo14.tgoVariant0
}

type ChoiceText struct{ Value string }

func NewChoiceText(tgo15 string) Choice {
	return Choice{tgoTag: ChoiceTagText, tgoVariant1: ChoiceText{tgo15}}
}
func (tgo16 Choice) TextPayload() ChoiceText {
	return tgo16.tgoVariant1
}

var calls int
var first = func() Choice {
	var payload_tgo1 ChoiceNumber = ChoiceNumber{Value: next()}
	var enum_tgo2 Choice = NewChoiceNumber(payload_tgo1.Value)
	return enum_tgo2
}()
var selected = func() int {
	var tgo3 int
	{
		var value_tgo4 Choice = first
		switch value_tgo4.Tag() {
		case ChoiceTagNumber:
			var value_tgo5 ChoiceNumber = value_tgo4.NumberPayload()
			tgo3 = value_tgo5.Value
		default:
			var value_tgo6 ChoiceText = value_tgo4.TextPayload()
			tgo3 = len(value_tgo6.Value)
		}
	}
	return tgo3
}()

var callback = func() string {
	{
		var payload_tgo7 ChoiceText = ChoiceText{Value: "ready"}
		var enum_tgo8 Choice = NewChoiceText(payload_tgo7.Value)
		var value_tgo9 Choice = (enum_tgo8)
		switch value_tgo9.Tag() {
		case ChoiceTagNumber:
			var value_tgo10 ChoiceNumber = value_tgo9.NumberPayload()
			return fmt.Sprint(value_tgo10.Value)
		default:
			var value_tgo11 ChoiceText = value_tgo9.TextPayload()
			return value_tgo11.Value
		}
	}

}

func next() int { calls++; return 7 }
func main()     { fmt.Println(selected, callback(), calls) }
