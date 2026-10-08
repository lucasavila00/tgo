package genericgood

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

func EmptyCounts[S countSlices]() S {
	return make(S, 0)
}

func Describe[E events](event E) string {
	switch event.TgoTag() {
	case 1:
		return event.TgoStarted().ID
	case 2:
		return event.TgoStopped().Reason
	default:
		panic("invalid Event variant")
	}
}
