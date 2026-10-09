package good

import (
	"example.com/tgolint/model"
	"example.com/tgolint/validated"
)

func validateEvent(event model.Event) (model.Event, error) {
	return event, nil
}

func CrossPackageValidation(event model.Event) string {
	value, err := validated.Event(event)
	if err != nil {
		return ""
	}
	switch value.TgoTag() {
	case 1:
		return value.TgoStarted().ID
	case 2:
		return value.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func CrossPackageConstructor() string {
	return Describe(validated.Constructed())
}

func ValidationFunctionValue(event model.Event) string {
	return Describe(event)
}

func ConstructorFunctionValue() string {
	construct := model.NewEventStopped
	return Describe(construct(model.EventStopped{}))
}

func TrustAssertion(input any) (model.Event, error) {
	return input.(model.Event), nil
}

func Describe(event model.Event) string {
	switch event.TgoTag() {
	case 1:
		started := event.TgoStarted()
		return started.ID
	case 2:
		stopped := event.TgoStopped()
		return stopped.Reason
	default:
		panic("invalid Event variant")
	}
}

func DescribeOrInvalid(event model.Event) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		return "invalid"
	}
}

func Values() ([]model.Event, error) {
	count, err := model.NewCount(1)
	if err != nil {
		return nil, err
	}
	_ = count.Value()
	return []model.Event{
		model.NewEventStarted(model.EventStarted{ID: "one"}),
	}, nil
}

func Find(values map[string]model.Event, key string) (model.Event, bool) {
	value, ok := values[key]
	return value, ok
}

func FindVar(values map[string]model.Event, key string) (model.Event, bool) {
	var value, ok = values[key]
	return value, ok
}

func KeepLength(values []model.Event) []model.Event {
	return values[:len(values)]
}

func Identity(event model.Event) model.Event {
	return model.Event(event)
}

type Envelope struct {
	Event model.Event
}

type Embedded struct {
	model.Event
}

func DescribeEnvelope(envelope Envelope) string {
	switch envelope.Event.TgoTag() {
	case 1:
		return envelope.Event.TgoStarted().ID
	case 2:
		return envelope.Event.TgoStopped().Reason
	default:
		for {
		}
	}
}

func DescribeEmbedded(embedded Embedded) string {
	switch embedded.TgoTag() {
	case 1:
		return embedded.TgoStarted().ID
	case 2:
		return embedded.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func CheckedGuard(input int) (int, error) {
	value, err := model.NewCount(input)
	if err != nil {
		return 0, err
	}
	return value.Value(), nil
}

func CheckedNilBranch(input int) (int, error) {
	value, err := model.NewCount(input)
	if err == nil {
		return value.Value(), nil
	}
	return 0, err
}

func CheckedReturn(input int) (model.Count, error) {
	return model.NewCount(input)
}

func CheckedPair(input int) (model.Count, error) {
	value, err := model.NewCount(input)
	return value, err
}

func countWrapper(input int) (model.Count, error) {
	return model.NewCount(input)
}

func CheckedWrapper(input int) (int, error) {
	value, err := countWrapper(input)
	if err != nil {
		return 0, err
	}
	return value.Value(), nil
}

func CheckedFunctionValue(input int) (int, error) {
	makeCount := model.NewCount
	value, err := makeCount(input)
	if err != nil {
		return 0, err
	}
	return value.Value(), nil
}

func CheckedCompound(input int, ready bool) (int, error) {
	value, err := countWrapper(input)
	if err == nil && ready {
		return value.Value(), nil
	}
	return 0, err
}

func CheckedLoop(input int, run bool) (int, error) {
	value, err := countWrapper(input)
	if err != nil {
		return 0, err
	}
	for run {
		value, err = countWrapper(input + 1)
		if err != nil {
			return 0, err
		}
		run = false
	}
	return value.Value(), nil
}

func CheckedClosure(input int) (int, error) {
	value, err := countWrapper(input)
	if err != nil {
		return 0, err
	}
	read := func() int {
		return value.Value()
	}
	return read(), nil
}

func CheckedGoto(input int) (int, error) {
	value, err := countWrapper(input)
	if err != nil {
		return 0, err
	}
	goto use
use:
	return value.Value(), nil
}

func CheckedBreak(input int, run bool) (int, error) {
	value, err := countWrapper(input)
	for run {
		if err != nil {
			return 0, err
		}
		break
	}
	if err != nil {
		return 0, err
	}
	return value.Value(), nil
}

func DeadBreak(input int) (int, error) {
	value, err := countWrapper(1)
	if err != nil {
		return 0, err
	}
	for input > 0 {
		value, err = countWrapper(input)
		break
	}
	value, err = countWrapper(1)
	if err != nil {
		return 0, err
	}
	return value.Value(), nil
}

func DeadContinue(input int, run bool) {
	value, err := countWrapper(input)
	for run {
		continue
		_, _ = value, err
	}
}

func DeadGoto(input int) (int, error) {
	value, err := countWrapper(input)
	goto use
use:
	value, err = countWrapper(1)
	if err != nil {
		return 0, err
	}
	return value.Value(), nil
}

func DeadFallthrough(input int) (int, error) {
	value, err := countWrapper(1)
	if err != nil {
		return 0, err
	}
	switch input {
	case 0:
		value, err = countWrapper(input)
		fallthrough
	case 1:
		value, err = countWrapper(1)
		if err != nil {
			return 0, err
		}
	}
	return value.Value(), nil
}

func CheckedParameters(
	value model.Count,
	err error,
	input int,
) (int, error) {
	value, err = countWrapper(input)
	if err != nil {
		return 0, err
	}
	return value.Value(), nil
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

type Events interface {
	model.Event
	TgoTag() uint8
	TgoStarted() model.EventStarted
	TgoStopped() model.EventStopped
}

func GenericSlice[S CountSlices]() S {
	return make(S, 0)
}

func GenericMap[M CountMaps](values M, key string) (model.Count, bool) {
	value, ok := values[key]
	return value, ok
}

func DescribeGeneric[E Events](event E) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func PresenceMap(values map[string]model.Event, key string) (model.Event, bool) {
	if value, ok := values[key]; ok {
		return value, true
	}
	return model.NewEventStopped(model.EventStopped{}), false
}

func PresenceChannel(values <-chan model.Event) (model.Event, bool) {
	value, ok := <-values
	if !ok {
		return model.NewEventStopped(model.EventStopped{}), false
	}
	return value, true
}

func PresenceAssertion(input any) (model.Event, bool) {
	value, ok := input.(model.Event)
	return value, ok
}

func eventWrapper(values map[string]model.Event, key string) (model.Event, bool) {
	value, ok := values[key]
	return value, ok
}

func PresenceWrapper(values map[string]model.Event, key string) string {
	value, ok := eventWrapper(values, key)
	if !ok {
		return ""
	}
	validated, err := validateEvent(value)
	if err != nil {
		return ""
	}
	value = validated
	switch value.TgoTag() {
	case 1:
		return value.TgoStarted().ID
	case 2:
		return value.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func PresenceBoolean(values map[string]model.Event, key string, ready bool) string {
	value, ok := eventWrapper(values, key)
	if ok == true && ready {
		validated, err := validateEvent(value)
		if err != nil {
			return ""
		}
		value = validated
		switch value.TgoTag() {
		case 1:
			return value.TgoStarted().ID
		case 2:
			return value.TgoStopped().Reason
		default:
			panic("invalid Event variant")
		}
	}
	return ""
}

func DescribeSnapshot(event model.Event) string {
	validated, err := validateEvent(event)
	if err != nil {
		return ""
	}
	event = validated
	switch snapshot := event; snapshot.TgoTag() {
	case 1:
		return snapshot.TgoStarted().ID
	case 2:
		return snapshot.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

func DescribeWithInternalBreak(event model.Event) string {
	validated, err := validateEvent(event)
	if err != nil {
		return ""
	}
	event = validated
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		for {
			break
		}
		panic("invalid Event variant")
	}
}

func DescribeWithInternalGoto(event model.Event) string {
	validated, err := validateEvent(event)
	if err != nil {
		return ""
	}
	event = validated
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		goto invalid
	invalid:
		panic("invalid Event variant")
	}
}
