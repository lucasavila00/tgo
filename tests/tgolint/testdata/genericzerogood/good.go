package genericzerogood

import (
	"example.com/tgolint/genericzero"
	"example.com/tgolint/genericzerowrap"
	"example.com/tgolint/model"
)

var savedNested = genericzero.Nested[int]()

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
		map[string]model.Event{"event": model.NewEventStarted("", "")},
		"event",
	)
	_ = genericzero.ChannelRead(make(chan model.Event))
	_ = genericzero.Assert[model.Event](model.NewEventStarted("", ""))
	_ = genericzero.Assert[model.Event](1)
	_ = genericzero.Reslice(
		[]model.Event{model.NewEventStarted("", "")},
		1,
	)
	_ = genericzero.MapDiscarded(
		map[string]model.Event{"event": model.NewEventStarted("", "")},
		"event",
	)
	_ = genericzero.ChannelDiscarded(make(chan model.Event))
	_ = genericzero.AssertDiscarded[model.Event](
		model.NewEventStarted("", ""),
	)
	genericzero.Maybe[model.Event](false)
	genericzero.Mutated[model.Event](true)
	genericzero.Copied[model.Event](false)
	genericzero.ReadByCall[model.Event](false)
	genericzero.Unless[model.Event](true)
	genericzero.Recursive[model.Event](false)
	genericzero.Never[model.Event]()
	genericzero.Dead[model.Event]()
	_ = genericzero.Nested[model.Event]()
	genericzerowrap.Maybe[model.Event](true)
	genericzerowrap.Make[model.Event](0)
	_ = genericzero.MaybeStarted(false, model.NewEventStarted("", ""))
	_ = genericzero.UnlessStarted(true, model.NewEventStarted("", ""))
	enabled := false
	genericzero.Maybe[model.Event](enabled)
	length := 0
	_ = genericzero.Make[model.Event](length)
	makeEvents := genericzero.Make[model.Event]
	_ = makeEvents(0)
	factory := genericzero.Factory[model.Event]{}
	makeMethod := factory.Make
	_ = makeMethod(0)
	genericzero.Nested[model.Event]()
	nested := genericzero.Nested[model.Event]()
	_ = nested
	genericzero.AliasedNested[int]()()
	genericzero.ForwardedNested[int]()()
	genericzero.GenericForwardedNested[int]()()
	genericzero.NestedGenericCall[int]()()
	genericzero.NestedReturnedGenericCall[int]()()
	genericzero.AliasedNestedReturnedGenericCall[int]()()
	genericzero.AliasedGenericValueCall[int]()()
	genericzero.TransitiveAliasedGenericValueCall[int]()()
	genericzero.TransitiveAliasedReturnedCall[int]()()
	genericzero.AssignedGenericValueCall[int]()()
	genericzero.AssignedReturnedCall[int]()()
	genericzero.CycledAliasedGenericValueCall[int]()()
	genericzero.OpaqueAliasedGenericValueCall[int](func() {})()
	genericzero.AmbiguousSafeGenericValueCall[model.Event]()()
	genericzero.DoubleNested[model.Event]()
	genericzero.DoubleNested[model.Event]()()
	genericzero.DoubleNested[int]()()()
	genericzero.TripleNested[model.Event]()
	genericzero.TripleNested[model.Event]()()
	genericzero.TripleNested[model.Event]()()()
	genericzero.TripleNested[int]()()()()
	genericzero.ParenthesizedAssignedNested[int]()()
	genericzero.ParenthesizedRangedAssignedNested[int]()()
	genericzero.SafeAliasedNested[model.Event]()()
	genericzero.SafeForwardedNested[model.Event]()()
	genericzero.SafeGenericForwardedNested[model.Event]()()
	genericzero.DiscardedNested[model.Event]()()
	genericzero.LocalLiteralContainer[int]()
	genericzero.LocalAssignedContainer[int]()
	_ = genericzero.ReturnedContainer[int]()
	_ = genericzero.ReturnedAnyContainer[int]()
	genericzero.EscapedAnyContainer[int]()
	genericzero.ForwardedLocalCall[int]()
	genericzero.ForwardedUnknownEscape[int]()
	genericzero.ConditionalForwardCall[model.Event](false)
	genericzero.ConditionalForwardCall[int](true)
	genericzero.NeverForwardCall[model.Event]()
	genericzero.DirectInvokedForwardCall[int]()
	genericzero.DiscardedForwardCapture[model.Event]()
	genericzero.DeadAlias[model.Event]()()
	genericzero.ConditionalAlias[model.Event](false)()
	genericzero.ConditionalAlias[int](true)()
	_ = genericzero.SafeContainerLength[model.Event]()
	genericzero.OverwrittenContainer[model.Event]()
	genericzero.UnusedCapturedClosure[model.Event]()
	genericzero.AlternativeNested[int, int](true)()
	genericzero.RangedAssignedNested[int]()()
	genericzero.RangedDefinedNested[int]()()
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
	partialFactory := genericzero.TripleNested[model.Event]
	partialFactory()
	partialFactory()()
	partialFactory()()()
	partialFirst := genericzero.TripleNested[model.Event]()
	partialFirst()
	partialFirst()()
	partialSecond := genericzero.TripleNested[model.Event]()()
	partialSecond()
	var initializedPartial = genericzero.TripleNested[model.Event]()()
	initializedPartial()
	transitiveFactory := genericzero.TripleNested[model.Event]
	transitiveFactorySecond := transitiveFactory
	transitiveFactorySecond()()()
	transitiveFactoryThird := transitiveFactorySecond
	transitiveFactoryThird()()()
	transitiveFirst := genericzero.TripleNested[model.Event]()
	var transitiveFirstSecond = transitiveFirst
	transitiveFirstSecond()()
	var transitiveFirstThird = transitiveFirstSecond
	transitiveFirstThird()()
	transitiveSecond := genericzero.TripleNested[model.Event]()()
	var transitiveSecondNext func() func()
	transitiveSecondNext = transitiveSecond
	transitiveSecondNext()
	var transitiveSecondLast func() func()
	transitiveSecondLast = transitiveSecondNext
	transitiveSecondLast()
	derivedFactory := genericzero.TripleNested[model.Event]
	derivedFirst := derivedFactory()
	derivedFirst()()
	var initializedDerivedFirst = derivedFactory()
	initializedDerivedFirst()()
	var assignedDerivedFirst func() func() func()
	assignedDerivedFirst = derivedFactory()
	assignedDerivedFirst()()
	transitiveDerivedFactory := derivedFactory
	transitiveDerivedFirst := transitiveDerivedFactory()
	transitiveDerivedFirst()()
	parenthesizedFactory := (genericzero.TripleNested[model.Event])
	parenthesizedFactory()()()
	parenthesizedFactoryAlias := (parenthesizedFactory)
	(parenthesizedFactoryAlias)()()()
	parenthesizedFirst := (parenthesizedFactory)()
	(parenthesizedFirst)()()
	genericzero.DirectLocalLiteralCall[int]()
	genericzero.ReturnedLocalLiteralCall[int]()()
	genericzero.ReturnPartialTriple[int]()()()
	_ = genericzero.CallAndReturnSafe[int]()
	genericzero.EscapeLiteral[int]()
	genericzero.EscapeNested[int]()
	genericzero.EscapeVariable[int]()
	_ = genericzero.EscapeAny[int]()
	_ = genericzero.EscapeContainer[int]()
}
