package genericzerogood

import (
	"example.com/tgolint/genericzero"
	"example.com/tgolint/genericzerowrap"
	"example.com/tgolint/model"
)

func Safe() {
	genericzero.Variable[int]()
	_ = genericzero.Named[int]()
	_ = genericzero.New[int]()
	_ = genericzero.Make[int](1)
	genericzero.Clear([]int{1})
	_ = genericzero.MapRead(map[string]int{"one": 1}, "one")
	_ = genericzero.ChannelRead(make(chan int))
	_ = genericzero.Assert[int](1)
	_ = genericzero.Reslice([]int{1}, 1)
	_ = genericzero.OmittedField[int]()
	_ = genericzero.OmittedArray[int]()
	_ = genericzero.OmittedSlice(1)
	genericzerowrap.Twice[int]()
	genericzero.Factory[int]{}.Variable()
	_ = genericzero.Make[model.Event](0)
	genericzero.Clear([]model.Event{})
	_ = genericzero.MapRead(
		map[string]model.Event{"event": model.NewEventStarted(model.EventStarted{})},
		"event",
	)
	_ = genericzero.ChannelRead(make(chan model.Event))
	_ = genericzero.Assert[model.Event](model.NewEventStarted(model.EventStarted{}))
	_ = genericzero.Assert[model.Event](1)
	_ = genericzero.Reslice(
		[]model.Event{model.NewEventStarted(model.EventStarted{})},
		1,
	)
	_ = genericzero.MapDiscarded(
		map[string]model.Event{"event": model.NewEventStarted(model.EventStarted{})},
		"event",
	)
	_ = genericzero.ChannelDiscarded(make(chan model.Event))
	_ = genericzero.AssertDiscarded[model.Event](
		model.NewEventStarted(model.EventStarted{}),
	)
	genericzero.Maybe[model.Event](false)
	genericzero.Unless[model.Event](true)
	genericzero.Recursive[model.Event](false)
	genericzero.Never[model.Event]()
	genericzero.Dead[model.Event]()
	_ = genericzero.Nested[model.Event]()
	genericzerowrap.Maybe[model.Event](true)
	genericzerowrap.Make[model.Event](0)
	_ = genericzero.MaybeStarted(false, model.NewEventStarted(model.EventStarted{}))
	_ = genericzero.UnlessStarted(true, model.NewEventStarted(model.EventStarted{}))
	makeEvents := genericzero.Make[model.Event]
	_ = makeEvents(0)
	factory := genericzero.Factory[model.Event]{}
	makeMethod := factory.Make
	_ = makeMethod(0)
	genericzero.Nested[model.Event]()
	nested := genericzero.Nested[model.Event]()
	_ = nested
}

func BoundaryAssertion(input any) {
	_ = genericzero.Assert[model.Event](input)
}

func CheckedPresence(event model.Event) {
	_ = genericzero.MapChecked(map[string]model.Event{"event": event}, "event")
	_ = genericzero.ChannelChecked(make(chan model.Event))
	_ = genericzero.AssertChecked[model.Event](event)
}

func Allowed[T any](value T) {
	_ = genericzero.EmptySlice[T]()
	_ = genericzero.CurrentSlice([]T{value})
	_ = genericzero.FullField(value)
	_ = genericzero.FullArray(value)
}

func FunctionValues() {
	variable := genericzero.Variable[int]
	variable()
	factory := genericzero.Factory[int]{}
	method := factory.Variable
	method()
}
