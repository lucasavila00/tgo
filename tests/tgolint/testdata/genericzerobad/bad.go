package genericzerobad

import (
	"example.com/tgolint/genericzero"
	"example.com/tgolint/genericzerowrap"
	"example.com/tgolint/model"
	"example.com/tgolint/othermodel"
)

var packageEnabled = false

type sameNamedEvents interface {
	model.Event | othermodel.Event
	Tag() model.EventTag
	UnknownTag() string
}

func MixedModels[T sameNamedEvents](event T) model.EventTag {
	return event.Tag()
}

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
	genericzero.AssignedTrue[model.Event](false)
	genericzero.Copied[model.Event](true)
	genericzero.ReadByCall[model.Event](true)
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

func LocalConstants() {
	enabled := true
	genericzero.Maybe[model.Event](enabled)
	length := 1
	_ = genericzero.Make[model.Event](length)
	_ = genericzero.Narrowed[model.Event](256)
	genericzero.ReceiverMutation[model.Event](false)
	genericzero.Maybe[model.Event](packageEnabled)
	enabled = false
	func() {
		genericzero.Maybe[model.Event](enabled)
	}()
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
	genericzero.AliasedNested[model.Event]()()
	genericzero.NamedAliasedNested[model.Event]()()
	genericzero.CapturedNested[model.Event]()()
	genericzero.AddressedNested[model.Event]()()
	genericzero.AddressedAlternativeNested[int, model.Event]()()
	genericzero.RangedAssignedNested[model.Event]()()
	genericzero.RangedDefinedNested[model.Event]()()
	genericzero.SelectedNested[model.Event]()()
	genericzero.IndexedNested[model.Event]()()
	genericzero.ForwardedNested[model.Event]()()
	genericzero.GenericForwardedNested[model.Event]()()
	genericzero.NestedGenericCall[model.Event]()()
	genericzero.NestedReturnedGenericCall[model.Event]()()
	genericzero.AliasedNestedReturnedGenericCall[model.Event]()()
	genericzero.AliasedGenericValueCall[model.Event]()()
	genericzero.ParenthesizedAssignedNested[model.Event]()()
	genericzero.ParenthesizedRangedAssignedNested[model.Event]()()
	genericzero.CalledForwardedNested[model.Event]()()
}

func ReturnedAlternatives(first bool) {
	genericzero.AlternativeNested[model.Event, int](first)()
	genericzero.AlternativeNested[int, model.Event](first)()
}

func UnresolvedEscape(forward func(func()) func()) {
	nested := genericzero.UnresolvedNested[model.Event](forward)
	keep(nested)
}

func keep(any) {}
