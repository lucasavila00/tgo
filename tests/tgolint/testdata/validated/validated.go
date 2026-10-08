package validated

import "example.com/tgolint/model"

func Event(value model.Event) (model.Event, error) {
	return model.ValidateEvent(value)
}

func Constructed() model.Event {
	return model.NewEventStopped(model.EventStopped{})
}
