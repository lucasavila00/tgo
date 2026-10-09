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
	Tag() model.EventTag
	UnknownTag() string
	StartedPayload() model.EventStarted
	StoppedPayload() model.EventStopped
}

func EmptyCounts[S countSlices]() S {
	return make(S, 0)
}

func Describe[E events](event E) string {
	switch event.Tag() {
	case model.EventTagZero:
		panic("zero Event")
	case model.EventTagStarted:
		return event.StartedPayload().ID
	case model.EventTagStopped:
		return event.StoppedPayload().Reason
	default:
		panic(event.UnknownTag()) // unreachable: tgolint requires a case per tag
	}
}
