package genericzerowrap

import (
	"example.com/tgolint/genericzero"
	"example.com/tgolint/model"
)

func Variable[T any]() {
	genericzero.Variable[T]()
}

func Twice[T any]() {
	Variable[T]()
}

func Maybe[T any](enabled bool) {
	genericzero.Maybe[T](!enabled)
}

func Make[T any](length int) {
	_ = genericzero.Make[T](length)
}

type eventLike interface {
	Tag() model.EventTag
	StartedPayload() model.EventStarted
}

func Started[T eventLike](event T) string {
	return genericzero.Started(event)
}
