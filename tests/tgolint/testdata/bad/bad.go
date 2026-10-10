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
	return value.StartedPayload().ID
}

func SingleResultCallback(decode func() model.Event) string {
	value := decode()
	return value.StartedPayload().ID
}

func TrustedBoundary(
	event model.Event,
	foreign func(model.Event) (model.Event, error),
) string {
	validate := foreign
	validate = foreign
	value, err := validate(event)
	if err != nil {
		return ""
	}
	return value.StartedPayload().ID
}

var zero model.Event

var publishedEvent = model.NewEventStopped("")
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
	_ = event.Tag()
	return event.StartedPayload().ID
}

func DirectPayloadMethod(event model.Event) func() model.EventStarted {
	return event.StartedPayload
}

func WrongEarlyExitPayload(event model.Event) string {
	if event.Tag() != model.EventTagStarted {
		return ""
	}
	return event.StoppedPayload().Reason
}

func AssignedEarlyExitPayload(event, replacement model.Event) string {
	if event.Tag() != model.EventTagStarted {
		return ""
	}
	event = replacement
	return event.StartedPayload().ID
}

func EscapedEarlyExitPayload(event model.Event) string {
	if event.Tag() != model.EventTagStarted {
		return ""
	}
	mutateEvent(&event)
	return event.StartedPayload().ID
}

func AssignedSwitchPayload(event, replacement model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		event = replacement
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return ""
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func ClosureSwitchPayload(event model.Event) func() string {
	switch event.Tag() {
	case model.EventTagStarted:
		return func() string { return event.StartedPayload().ID }
	case model.EventTagStopped:
		return nil
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

type EventView interface {
	Tag() model.EventTag
	UnknownTag() string
	StartedPayload() model.EventStarted
	StoppedPayload() model.EventStopped
}

func InterfaceAccessor(event model.Event) string {
	var view EventView = event
	return view.StartedPayload().ID
}

type EventAccess interface {
	StartedPayload() model.EventStarted
}

func StructuralGeneric[T EventAccess](event T) string {
	return event.StartedPayload().ID
}

type TagView interface {
	Tag() model.EventTag
	UnknownTag() string
}

func InterfaceTag(event model.Event) model.EventTag {
	var view TagView = event
	return view.Tag()
}

func Incomplete(event model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		fallthrough
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	}
	return ""
}

func NamedResult() (event model.Event) {
	return
}

func EmptyDefault(event model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
	}
	return ""
}

func EscapingDefault(event model.Event, escape bool) string {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		if escape {
			break
		}
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
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
	return model.NewEventStopped(""), nil
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
	value := model.NewEventStopped("")
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
	switch envelope.Event.Tag() {
	case model.EventTagStarted:
		envelope.Event = model.NewEventStopped("changed")
		return envelope.Event.StartedPayload().ID
	case model.EventTagStopped:
		return envelope.Event.StoppedPayload().Reason
	default:
		panic(envelope.Event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func WrongPromoted(embedded Embedded) string {
	switch embedded.Tag() {
	case model.EventTagStarted:
		return embedded.StoppedPayload().Reason
	case model.EventTagStopped:
		return embedded.StoppedPayload().Reason
	default:
		panic(embedded.UnknownTag()) // unreachable: tgolint requires a case per tag
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
	Tag() model.EventTag
	UnknownTag() string
	StartedPayload() model.EventStarted
	StoppedPayload() model.EventStopped
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
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StoppedPayload().Reason
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func GotoDefault(event model.Event, escape bool) string {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		if escape {
			goto done
		}
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
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
	result := value.StartedPayload().ID
	if !ok {
		return ""
	}
	return result
}

func PresenceChannelEarly(values <-chan model.Event) string {
	value, ok := <-values
	_ = value.StartedPayload()
	if !ok {
		return ""
	}
	return ""
}

func PresenceAssertionEarly(input any) string {
	value, ok := input.(model.Event)
	result := value.StartedPayload().ID
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
	return value.StartedPayload().ID
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
	value := model.NewEventStopped("")
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
	return model.NewEventStopped("")
}

func PointerReceiver(event *model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func mutateEvent(event *model.Event) {
	*event = model.NewEventStopped("")
}

func AddressedReceiver(event model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		mutateEvent(&event)
		return event.StartedPayload().ID
	case model.EventTagStopped:
		mutateEvent(&event)
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func AliasedReceiver(event model.Event) string {
	pointer := &event
	switch event.Tag() {
	case model.EventTagStarted:
		*pointer = model.NewEventStopped("")
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func DelayedClosure(event model.Event) func() string {
	switch event.Tag() {
	case model.EventTagStarted:
		return func() string { return event.StartedPayload().ID }
	case model.EventTagStopped:
		return func() string { return event.StoppedPayload().Reason }
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func useEventStarted(model.EventStarted) {}

func DelayedDefer(event model.Event) {
	switch event.Tag() {
	case model.EventTagStarted:
		defer useEventStarted(event.StartedPayload())
		return
	case model.EventTagStopped:
		_ = event.StoppedPayload()
		return
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func DelayedGo(event model.Event) {
	switch event.Tag() {
	case model.EventTagStarted:
		go useEventStarted(event.StartedPayload())
		return
	case model.EventTagStopped:
		_ = event.StoppedPayload()
		return
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func CapturedReceiver(event model.Event) string {
	mutate := func() {
		event = model.NewEventStopped("")
	}
	switch event.Tag() {
	case model.EventTagStarted:
		mutate()
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

type MutableEmbedded struct {
	model.Event
}

func (event *MutableEmbedded) stop() {
	event.Event = model.NewEventStopped("")
}

func PointerMethodMutation(event MutableEmbedded) string {
	switch event.Tag() {
	case model.EventTagStarted:
		event.stop()
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func OuterCapture(
	event model.Event,
) (func() string, func()) {
	read := func() string {
		switch event.Tag() {
		case model.EventTagStarted:
			return event.StartedPayload().ID
		case model.EventTagStopped:
			return event.StoppedPayload().Reason
		default:
			panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
		}
	}
	write := func() {
		event = model.NewEventStopped("")
	}
	return read, write
}

func ReceiverLoop(event model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		for range 2 {
			_ = event.StartedPayload()
			event = model.NewEventStopped("")
		}
		return ""
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func ReceiverRange(event model.Event, events []model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		for _, event = range events {
			_ = event.StartedPayload()
		}
		return ""
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func ReceiverGoto(event model.Event, repeat bool) string {
	switch event.Tag() {
	case model.EventTagStarted:
	again:
		result := event.StartedPayload().ID
		event = model.NewEventStopped("")
		if repeat {
			repeat = false
			goto again
		}
		return result
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

type PointerEmbedded struct {
	*model.Event
}

func PointerField(event PointerEmbedded) string {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

const unrelatedStarted model.EventTag = 1

func NumericTag(event model.Event) string {
	switch event.Tag() {
	case 0:
		return "zero"
	case model.EventTagStarted:
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func UnrelatedTag(event model.Event) string {
	switch event.Tag() {
	case unrelatedStarted:
		return "started"
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func MultiTagPayload(event model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted, model.EventTagStopped:
		return event.StartedPayload().ID
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func PayloadMethodValue(event model.Event) func() model.EventStarted {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StartedPayload
	case model.EventTagStopped:
		return nil
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func MissingTag(event model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StartedPayload().ID
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func WrongGoPayload(event model.Event) {
	switch event.Tag() {
	case model.EventTagStarted:
		return
	case model.EventTagStopped:
		go useEventStarted(event.StartedPayload())
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func WrongDeferPayload(event model.Event) {
	switch event.Tag() {
	case model.EventTagStarted:
		return
	case model.EventTagStopped:
		defer useEventStarted(event.StartedPayload())
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func DefaultPayload(event model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StartedPayload().ID
	default:
		return event.StartedPayload().ID
	}
}

func LiteralHoles() {
	_ = [2]model.Event{0: model.NewEventStopped("")}
	_ = []model.Event{1: model.NewEventStopped("")}
}

func BypassedEarlyExitPayload(event model.Event) string {
	goto payload
	if event.Tag() != model.EventTagStarted {
		return ""
	}
payload:
	return event.StartedPayload().ID
}

func EscapedReceiverLoop(event model.Event) string {
	switch event.Tag() {
	case model.EventTagStarted:
		for range 2 {
			_ = event.StartedPayload()
			mutateEvent(&event)
		}
		return ""
	case model.EventTagStopped:
		return ""
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func EscapedReceiverGoto(event model.Event, repeat bool) string {
	if event.Tag() != model.EventTagStarted {
		return ""
	}
again:
	result := event.StartedPayload().ID
	mutateEvent(&event)
	if repeat {
		repeat = false
		goto again
	}
	return result
}

func EmbeddedAssignment(event MutableEmbedded) string {
	switch event.Tag() {
	case model.EventTagStarted:
		event.Event = model.NewEventStopped("")
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return ""
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}

func PackageReceiverPayload() string {
	if publishedEvent.Tag() != model.EventTagStopped {
		return ""
	}
	return publishedEvent.StoppedPayload().Reason
}

func InsufficientIfProof(event model.Event, ready bool) string {
	if ready || event.Tag() == model.EventTagStarted {
		return event.StartedPayload().ID
	}
	return ""
}

func InsufficientEarlyExitProof(event model.Event, ready bool) string {
	if ready && event.Tag() != model.EventTagStarted {
		return ""
	}
	return event.StartedPayload().ID
}

func ChangeNested(value *model.Nested) {
	value.Number = 1
	value.Values[0] = 2
	value.Number++
	value.Values[:][0] = 3
	for value.Values[1] = range []int{3} {
	}
	_ = &value.Values[0]
	value.Pointer = nil
	value.ValueMiddle.Number = 4
	value.ValueArray[0].Number = 5
}
