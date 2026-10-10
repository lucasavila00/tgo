package genericzerobad

import (
	"example.com/tgolint/genericzero"
	"example.com/tgolint/genericzerowrap"
	"example.com/tgolint/model"
	"example.com/tgolint/othermodel"
)

var packageEnabled = false
var savedNested = genericzero.Nested[model.Event]()

type sameNamedEvents interface {
	model.Event | othermodel.Event
	Tag() model.EventTag
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
	genericzero.Factory[genericzero.Slot[model.Event]]{}.ReturnedVariable()()
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
	genericzero.NamedCapturedCellBeforeWrite[model.Event]()()
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
	genericzero.TransitiveAliasedGenericValueCall[model.Event]()()
	genericzero.TransitiveAliasedReturnedCall[model.Event]()()
	genericzero.AssignedGenericValueCall[model.Event]()()
	genericzero.AssignedReturnedCall[model.Event]()()
	genericzero.CycledAliasedGenericValueCall[model.Event]()()
	genericzero.OpaqueAliasedGenericValueCall[model.Event](func() {})()
	genericzero.DoubleNested[model.Event]()()()
	genericzero.TripleNested[model.Event]()()()()
	tripleFactory := genericzero.TripleNested[model.Event]
	tripleFactory()()()()
	tripleFirst := genericzero.TripleNested[model.Event]()
	tripleFirst()()()
	tripleSecond := genericzero.TripleNested[model.Event]()()
	tripleSecond()()
	var initializedTripleSecond = genericzero.TripleNested[model.Event]()()
	initializedTripleSecond()()
	transitiveFactory := genericzero.TripleNested[model.Event]
	transitiveFactorySecond := transitiveFactory
	transitiveFactorySecond()()()()
	transitiveFactoryThird := transitiveFactorySecond
	transitiveFactoryThird()()()()
	transitiveFirst := genericzero.TripleNested[model.Event]()
	var transitiveFirstSecond = transitiveFirst
	transitiveFirstSecond()()()
	var transitiveFirstThird = transitiveFirstSecond
	transitiveFirstThird()()()
	transitiveSecond := genericzero.TripleNested[model.Event]()()
	var transitiveSecondNext func() func()
	transitiveSecondNext = transitiveSecond
	transitiveSecondNext()()
	var transitiveSecondLast func() func()
	transitiveSecondLast = transitiveSecondNext
	transitiveSecondLast()()
	derivedFactory := genericzero.TripleNested[model.Event]
	derivedFirst := derivedFactory()
	derivedFirst()()()
	var initializedDerivedFirst = derivedFactory()
	initializedDerivedFirst()()()
	var assignedDerivedFirst func() func() func()
	assignedDerivedFirst = derivedFactory()
	assignedDerivedFirst()()()
	transitiveDerivedFactory := derivedFactory
	transitiveDerivedFirst := transitiveDerivedFactory()
	transitiveDerivedFirst()()()
	parenthesizedFactory := (genericzero.TripleNested[model.Event])
	parenthesizedFactory()()()()
	parenthesizedFactoryAlias := (parenthesizedFactory)
	(parenthesizedFactoryAlias)()()()()
	parenthesizedFirst := (parenthesizedFactory)()
	(parenthesizedFirst)()()()
	genericzero.DirectLocalLiteralCall[model.Event]()
	genericzero.ReturnedLocalLiteralCall[model.Event]()()
	genericzero.ReturnPartialTriple[model.Event]()()()
	_ = genericzero.CallAndReturnSafe[model.Event]()
	genericzero.EscapeLiteral[model.Event]()
	genericzero.EscapeNested[model.Event]()
	genericzero.EscapeVariable[model.Event]()
	_ = genericzero.EscapeAny[model.Event]()
	_ = genericzero.EscapeContainer[model.Event]()
	genericzero.ParenthesizedAssignedNested[model.Event]()()
	genericzero.ParenthesizedRangedAssignedNested[model.Event]()()
	genericzero.CalledForwardedNested[model.Event]()()
	genericzero.LocalLiteralContainer[model.Event]()
	genericzero.LocalAssignedContainer[model.Event]()
	_ = genericzero.ReturnedContainer[model.Event]()
	_ = genericzero.ReturnedAnyContainer[model.Event]()
	genericzero.EscapedAnyContainer[model.Event]()
	genericzero.ForwardedLocalCall[model.Event]()
	genericzero.ForwardedUnknownEscape[model.Event]()
	genericzero.ConditionalForwardCall[model.Event](true)
	genericzero.DirectInvokedForwardCall[model.Event]()
	genericzero.ConditionalAlias[model.Event](true)()
	_ = genericzero.ReturnAssignedBox[model.Event]()
	_ = genericzero.ReturnedAppendedAlias[model.Event]()
	genericzero.CopyAlias[model.Event](make([]func(), 1))
	genericzero.OrderedUnsafe[model.Event]()
	genericzero.LoadedBeforeWrite[model.Event]()
	genericzero.CapturedCellBeforeWrite[model.Event]()()
	genericzero.SharedSlotAfterWrite[model.Event]()
	genericzero.DiscardedAppendReuse[model.Event]()
	genericzero.CopyThenCall[model.Event]()
	genericzero.RecursiveCopyThenCall[model.Event]()
	genericzero.RecursiveUnsafe[model.Event]()
	genericzero.GuardedUnsafe[model.Event]()
	genericzero.DistinctActualsUnsafe[model.Event]()
	genericzero.MultipleAssignmentUnsafe[model.Event]()
	genericzero.NestedArgumentUnsafe[model.Event]()
	imported := []func(){genericzero.Nested[model.Event]()}
	genericzerowrap.CallThenStore(imported, func() {})
	imported = []func(){genericzero.Nested[model.Event]()}
	genericzerowrap.LoadThenStoreCall(imported, func() {})
	first := []func(){genericzero.Nested[model.Event]()}
	second := []func(){genericzero.Nested[model.Event]()}
	genericzerowrap.StoreFirstCallSecond(first, second)
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
