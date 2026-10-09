package validated

import "example.com/tgolint/model"

func Event(value model.Event) (model.Event, error) {
	return value, nil
}

func Constructed() model.Event {
	return model.EventStopped{}.Event()
}
