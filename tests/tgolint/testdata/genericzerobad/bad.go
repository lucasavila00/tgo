package genericzerobad

import (
	"example.com/tgolint/genericzero"
	"example.com/tgolint/genericzerowrap"
	"example.com/tgolint/model"
)

func Direct(
	event model.Event,
	values []model.Event,
	mapping map[string]model.Event,
	key string,
	channel <-chan model.Event,
	input any,
	length int,
) {
	genericzero.Variable[model.Event]()
	_ = genericzero.Named[model.Event]()
	_ = genericzero.New[model.Event]()
	_ = genericzero.Make[model.Event](1)
	genericzero.Clear([]model.Event{event})
	genericzero.Clear(values)
	_ = genericzero.MapRead(map[string]model.Event{}, "event")
	_ = genericzero.MapRead(mapping, key)
	_ = genericzero.ChannelRead(channel)
	_ = genericzero.Assert[model.Event](input)
	_ = genericzero.MapDiscarded(map[string]model.Event{}, "event")
	_ = genericzero.MapDiscarded(mapping, key)
	_ = genericzero.ChannelDiscarded(channel)
	_ = genericzero.AssertDiscarded[model.Event](input)
	_ = genericzero.AssertDiscarded[model.Event](1)
	_ = genericzero.Reslice(values, length)
	_ = genericzero.Reslice(make([]model.Event, 0, 1), 1)
	_ = genericzero.OmittedField[model.Event]()
	_ = genericzero.OmittedArray[model.Event]()
	_ = genericzero.OmittedSlice(event)
	genericzero.Maybe[model.Event](true)
	genericzero.Mutated[model.Event](true)
	genericzero.Mutated[model.Event](false)
	genericzero.Unless[model.Event](false)
	genericzero.Recursive[model.Event](true)
	_ = genericzero.MaybeStarted(true, event)
	_ = genericzero.UnlessStarted(false, event)
}

func Conditional(event model.Event, enabled bool, length int) {
	genericzero.Maybe[model.Event](enabled)
	genericzero.Unless[model.Event](enabled)
	genericzero.Recursive[model.Event](enabled)
	_ = genericzero.Make[model.Event](length)
	_ = genericzero.MaybeStarted(enabled, event)
	_ = genericzero.UnlessStarted(enabled, event)
	genericzerowrap.Maybe[model.Event](enabled)
	genericzerowrap.Make[model.Event](length)
}

func Wrapped(event model.Event) {
	genericzerowrap.Twice[model.Event]()
	_ = genericzero.Started(event)
	_ = genericzerowrap.Started(event)
	genericzero.Factory[model.Event]{}.Variable()
	_ = genericzero.Reader[model.Event]{}.Started(event)
}

func FunctionValues(event model.Event) {
	variable := genericzero.Variable[model.Event]
	variable()
	makeEvents := genericzero.Make[model.Event]
	_ = makeEvents(1)
	started := genericzero.Started[model.Event]
	_ = started(event)

	factory := genericzero.Factory[model.Event]{}
	method := factory.Variable
	method()
	makeMethod := factory.Make
	_ = makeMethod(1)
	reader := genericzero.Reader[model.Event]{}
	read := reader.Started
	_ = read(event)

	escaped := genericzero.Make[model.Event]
	keep(escaped)
	genericzero.Nested[model.Event]()()
	nested := genericzero.Nested[model.Event]()
	nested()
}

func keep(any) {}
