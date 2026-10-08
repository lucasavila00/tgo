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
	TgoTag() uint8
	TgoStarted() model.EventStarted
	TgoStopped() model.EventStopped
}

func FilledCounts[S countSlices]() S {
	return make(S, 1)
}

func ZeroEvent[E events]() {
	var event E
	_ = event
}

func WrongPayload[E events](event E) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStopped().Reason
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

type Decoy struct{}

func (Decoy) TgoTag() uint8 {
	return 1
}

func (Decoy) TgoStarted() model.EventStarted {
	return model.EventStarted{}
}

func (Decoy) TgoStopped() model.EventStopped {
	return model.EventStopped{}
}

type mixedEvents interface {
	model.Event | Decoy
	TgoTag() uint8
	TgoStarted() model.EventStarted
	TgoStopped() model.EventStopped
}

func MixedDirect[E mixedEvents](event E) string {
	return event.TgoStarted().ID
}

type eventPointers interface {
	*model.Event
	TgoTag() uint8
	TgoStarted() model.EventStarted
	TgoStopped() model.EventStopped
}

func PointerGeneric[E eventPointers](event E) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}

type differentModels interface {
	model.Event | model.Signal
	TgoTag() uint8
}

func MixedModels[M differentModels](value M) uint8 {
	return value.TgoTag()
}
