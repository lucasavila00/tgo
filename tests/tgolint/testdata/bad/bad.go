package bad

import "example.com/tgolint/model"

func AssertExisting(input any) model.Event {
	return input.(model.Event)
}

func CallbackBoundary(decode func() (model.Event, error)) string {
	value, err := decode()
	if err != nil {
		return ""
	}
	return value.TgoStarted().ID
}

func SingleResultCallback(decode func() model.Event) string {
	value := decode()
	return value.TgoStarted().ID
}

func ReassignedValidator(
	event model.Event,
	foreign func(model.Event) (model.Event, error),
) string {
	validate := model.ValidateEvent
	validate = foreign
	value, err := validate(event)
	if err != nil {
		return ""
	}
	return value.TgoStarted().ID
}

var zero model.Event

var publishedEvent = model.NewEventStopped(model.EventStopped{})
var publishedError error
var publishedOK bool

type Outer struct {
	Event model.Event
}

type EventClone model.Event

func InvalidValues(values map[string]model.Event, slice []model.Event, key string) {
	_ = model.Event{}
	_ = Outer{}
	_ = new(model.Count)
	_ = make([]model.Event, 1)
	_ = values[key]
	_ = slice[:cap(slice)]
}

func Direct(event model.Event) string {
	_ = event.TgoTag()
	return event.TgoStarted().ID
}

type EventView interface {
	TgoTag() uint8
	TgoStarted() model.EventStarted
	TgoStopped() model.EventStopped
}

func InterfaceAccessor(event model.Event) string {
	var view EventView = event
	return view.TgoStarted().ID
}

type EventAccess interface {
	TgoStarted() model.EventStarted
}

func StructuralGeneric[T EventAccess](event T) string {
	return event.TgoStarted().ID
}

type TagView interface {
	TgoTag() uint8
}

func InterfaceTag(event model.Event) uint8 {
	var view TagView = event
	return view.TgoTag()
}

func Incomplete(event model.Event) string {
	switch event.TgoTag() {
	case 1:
		fallthrough
	case 2:
		return event.TgoStopped().Reason
	}
	return ""
}

func NamedResult() (event model.Event) {
	return
}

func EmptyDefault(event model.Event) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
	}
	return ""
}

func EscapingDefault(event model.Event, escape bool) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		if escape {
			break
		}
		panic("invalid Event variant")
	}
	return ""
}

func DiscardConstructor(input int) {
	_, _ = model.NewCount(input)
}

func DiscardConstructorCall(input int) {
	model.NewCount(input)
}

func countWrapper(input int) (model.Count, error) {
	return model.NewCount(input)
}

func DiscardWrapperError(input int) int {
	value, _ := countWrapper(input)
	return value.Value()
}

func DiscardFunctionError(input int) int {
	makeCount := model.NewCount
	value, _ := makeCount(input)
	return value.Value()
}

func eventWithError() (model.Event, error) {
	return model.NewEventStopped(model.EventStopped{}), nil
}

func DiscardEnumError() model.Event {
	value, _ := eventWithError()
	return value
}

func PublishResult() {
	publishedEvent, publishedError = eventWithError()
	if publishedError == nil {
		_ = publishedEvent
	}
}

func PublishPresence(values map[string]model.Event, key string) {
	publishedEvent, publishedOK = values[key]
	if publishedOK {
		_ = publishedEvent
	}
}

func CaptureResult() func() {
	value := model.NewEventStopped(model.EventStopped{})
	var err error
	return func() {
		value, err = eventWithError()
		if err == nil {
			_ = value
		}
	}
}

func UseBeforeCheck(input int) int {
	value, err := model.NewCount(input)
	_ = err
	return value.Value()
}

func DeclarationBeforeCheck(input int) int {
	value, err := model.NewCount(input)
	var number = value.Value()
	if err != nil {
		return 0
	}
	return number
}

func OverwriteError(input int) int {
	first, err := model.NewCount(input)
	second, err := model.NewCount(input + 1)
	if err == nil {
		_ = second.Value()
		return first.Value()
	}
	return 0
}

func ShadowNil(input int, nil error) int {
	value, err := model.NewCount(input)
	if err == nil {
		return value.Value()
	}
	return 0
}

func RangeOverwritesError(input int) int {
	value, err := model.NewCount(input)
	for _, err = range []error{nil} {
	}
	if err == nil {
		return value.Value()
	}
	return 0
}

type Envelope struct {
	Event model.Event
}

type Embedded struct {
	model.Event
}

func ChangedReceiver(envelope Envelope) string {
	switch envelope.Event.TgoTag() {
	case 1:
		envelope.Event = model.NewEventStopped(model.EventStopped{Reason: "changed"})
		return envelope.Event.TgoStarted().ID
	case 2:
		return envelope.Event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func WrongPromoted(embedded Embedded) string {
	switch embedded.TgoTag() {
	case 1:
		return embedded.TgoStopped().Reason
	case 2:
		return embedded.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

type Counts interface {
	model.Count
}

type CountSlices interface {
	~[]model.Count
}

type CountMaps interface {
	~map[string]model.Count
}

type CountChannels interface {
	~chan model.Count
}

type Events interface {
	model.Event
	TgoTag() uint8
	TgoStarted() model.EventStarted
	TgoStopped() model.EventStopped
}

func GenericZero[T Counts]() {
	var value T
	_ = value
}

func GenericNamed[T Counts]() (value T) {
	return
}

func GenericNew[T Counts]() *T {
	return new(T)
}

func GenericMake[S CountSlices]() S {
	return make(S, 1)
}

func GenericClear[S CountSlices](values S) {
	clear(values)
}

func GenericMap[M CountMaps](values M, key string) model.Count {
	return values[key]
}

func GenericChannel[C CountChannels](values C) model.Count {
	return <-values
}

func GenericAssert[T Counts](value any) T {
	return value.(T)
}

func GenericWrongAccessor[E Events](event E) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStopped().Reason
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func GotoDefault(event model.Event, escape bool) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		if escape {
			goto done
		}
		panic("invalid Event variant")
	}
done:
	return ""
}

func PresenceMapBlank(values map[string]model.Event, key string) {
	value, _ := values[key]
	_ = value
}

func PresenceMapEarly(values map[string]model.Event, key string) string {
	value, ok := values[key]
	result := value.TgoStarted().ID
	if !ok {
		return ""
	}
	return result
}

func PresenceChannelEarly(values <-chan model.Event) string {
	value, ok := <-values
	_ = value.TgoStarted()
	if !ok {
		return ""
	}
	return ""
}

func PresenceAssertionEarly(input any) string {
	value, ok := input.(model.Event)
	result := value.TgoStarted().ID
	if !ok {
		return ""
	}
	return result
}

func eventWrapper(values map[string]model.Event, key string) (model.Event, bool) {
	value, ok := values[key]
	return value, ok
}

func PresenceWrapperBlank(values map[string]model.Event, key string) string {
	value, _ := eventWrapper(values, key)
	return value.TgoStarted().ID
}

func LoopOverwrite(input int, run bool) int {
	value, err := model.NewCount(1)
	if err != nil {
		return 0
	}
	for run {
		value, err = model.NewCount(input)
		_ = err
		run = false
	}
	return value.Value()
}

func SwitchOverwrite(input int) int {
	value, err := model.NewCount(1)
	if err != nil {
		return 0
	}
	switch input {
	case 1:
		value, err = model.NewCount(input)
		_ = err
	}
	return value.Value()
}

func FallthroughOverwrite(input int) int {
	value, err := model.NewCount(1)
	if err != nil {
		return 0
	}
	switch input {
	case 0:
		value, err = model.NewCount(input)
		_ = err
		fallthrough
	case 1:
		return value.Value()
	}
	return 0
}

func TypeSwitchOverwrite(input any) int {
	value, err := model.NewCount(1)
	if err != nil {
		return 0
	}
	switch input.(type) {
	case int:
		value, err = model.NewCount(2)
		_ = err
	}
	return value.Value()
}

func GotoSkipsError(input int) int {
	value, err := model.NewCount(input)
	goto use
	if err != nil {
		return 0
	}
use:
	return value.Value()
}

func GotoRepeatsUnchecked(input int, again bool) int {
	value, err := model.NewCount(1)
	if err != nil {
		return 0
	}
again:
	result := value.Value()
	value, err = model.NewCount(input)
	_ = err
	if again {
		again = false
		goto again
	}
	return result
}

func BreakSkipsError(input int) int {
	value, err := model.NewCount(input)
	switch input {
	case 1:
		break
		if err != nil {
			return 0
		}
	default:
		return 0
	}
	return value.Value()
}

func ContinueSkipsError(input int, run bool) int {
	value, err := model.NewCount(input)
	for run {
		run = false
		continue
		if err != nil {
			return 0
		}
	}
	return value.Value()
}

func SelectOverwrite(values <-chan model.Event) model.Event {
	value := model.NewEventStopped(model.EventStopped{})
	var ok bool
	select {
	case value, ok = <-values:
		_ = ok
	}
	return value
}

func ErrorAliasBefore(input int) int {
	var err error
	pointer := &err
	value, err := model.NewCount(input)
	*pointer = nil
	if err == nil {
		return value.Value()
	}
	return 0
}

func ErrorAliasAfter(input int) int {
	value, err := model.NewCount(input)
	*(&err) = nil
	if err == nil {
		return value.Value()
	}
	return 0
}

func PresenceAliasBefore(values map[string]model.Event, key string) model.Event {
	var ok bool
	pointer := &ok
	value, ok := values[key]
	*pointer = true
	if ok {
		return value
	}
	return model.NewEventStopped(model.EventStopped{})
}

func PointerReceiver(event *model.Event) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func mutateEvent(event *model.Event) {
	*event = model.NewEventStopped(model.EventStopped{})
}

func AddressedReceiver(event model.Event) string {
	switch event.TgoTag() {
	case 1:
		mutateEvent(&event)
		return event.TgoStarted().ID
	case 2:
		mutateEvent(&event)
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func AliasedReceiver(event model.Event) string {
	pointer := &event
	switch event.TgoTag() {
	case 1:
		*pointer = model.NewEventStopped(model.EventStopped{})
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func DelayedClosure(event model.Event) func() string {
	switch event.TgoTag() {
	case 1:
		return func() string { return event.TgoStarted().ID }
	case 2:
		return func() string { return event.TgoStopped().Reason }
	default:
		panic("invalid Event variant")
	}
}

func useEventStarted(model.EventStarted) {}

func DelayedDefer(event model.Event) {
	switch event.TgoTag() {
	case 1:
		defer useEventStarted(event.TgoStarted())
		return
	case 2:
		_ = event.TgoStopped()
		return
	default:
		panic("invalid Event variant")
	}
}

func DelayedGo(event model.Event) {
	switch event.TgoTag() {
	case 1:
		go useEventStarted(event.TgoStarted())
		return
	case 2:
		_ = event.TgoStopped()
		return
	default:
		panic("invalid Event variant")
	}
}

func CapturedReceiver(event model.Event) string {
	mutate := func() {
		event = model.NewEventStopped(model.EventStopped{})
	}
	switch event.TgoTag() {
	case 1:
		mutate()
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

type MutableEmbedded struct {
	model.Event
}

func (event *MutableEmbedded) stop() {
	event.Event = model.NewEventStopped(model.EventStopped{})
}

func PointerMethodMutation(event MutableEmbedded) string {
	switch event.TgoTag() {
	case 1:
		event.stop()
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func OuterCapture(
	event model.Event,
) (func() string, func()) {
	read := func() string {
		switch event.TgoTag() {
		case 1:
			return event.TgoStarted().ID
		case 2:
			return event.TgoStopped().Reason
		default:
			panic("invalid Event variant")
		}
	}
	write := func() {
		event = model.NewEventStopped(model.EventStopped{})
	}
	return read, write
}

func ReceiverLoop(event model.Event) string {
	switch event.TgoTag() {
	case 1:
		for range 2 {
			_ = event.TgoStarted()
			event = model.NewEventStopped(model.EventStopped{})
		}
		return ""
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func ReceiverRange(event model.Event, events []model.Event) string {
	switch event.TgoTag() {
	case 1:
		for _, event = range events {
			_ = event.TgoStarted()
		}
		return ""
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func ReceiverGoto(event model.Event, repeat bool) string {
	switch event.TgoTag() {
	case 1:
	again:
		result := event.TgoStarted().ID
		event = model.NewEventStopped(model.EventStopped{})
		if repeat {
			repeat = false
			goto again
		}
		return result
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

type PointerEmbedded struct {
	*model.Event
}

func PointerField(event PointerEmbedded) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}
