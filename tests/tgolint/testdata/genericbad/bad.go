package genericbad

import "example.com/tgolint/model"

type CountSlice []model.Count

type sliceChoices interface {
	CountSlice | string
}

type countSlices interface {
	~[]model.Count
	sliceChoices
}

type eventChoices interface {
	model.Event | int
}

type events interface {
	eventChoices
	model.Event
	Tag() model.EventTag
	StartedPayload() model.EventStarted
	StoppedPayload() model.EventStopped
}

func FilledCounts[S countSlices]() S {
	return make(S, 1)
}

func ZeroEvent[E events]() {
	var event E
	_ = event
}

func WrongPayload[E events](event E) string {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StoppedPayload().Reason
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic("invalid Event tag") // unreachable: tgolint requires a case per tag
	}
}

type Decoy struct{}

func (Decoy) Tag() model.EventTag {
	return model.EventTagStarted
}

func (Decoy) StartedPayload() model.EventStarted {
	return model.EventStarted{}
}

func (Decoy) StoppedPayload() model.EventStopped {
	return model.EventStopped{}
}

type mixedEvents interface {
	model.Event | Decoy
	Tag() model.EventTag
	StartedPayload() model.EventStarted
	StoppedPayload() model.EventStopped
}

func MixedDirect[E mixedEvents](event E) string {
	return event.StartedPayload().ID
}

type eventPointers interface {
	*model.Event
	Tag() model.EventTag
	StartedPayload() model.EventStarted
	StoppedPayload() model.EventStopped
}

func PointerGeneric[E eventPointers](event E) string {
	switch event.Tag() {
	case model.EventTagStarted:
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic("invalid Event tag") // unreachable: tgolint requires a case per tag
	}
}

type differentModels interface {
	model.Event | model.Signal
	Tag() model.EventTag
}

func MixedModels[M differentModels](value M) model.EventTag {
	return value.Tag()
}
