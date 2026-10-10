package structuralprojectiongood

import (
	"example.com/tgolint/model"
	"example.com/tgolint/structuralprojection"
)

type localFactory[T any] struct{}

func localZero[T any]() {
	var value T
	_ = value
}

func localSafe[A, B any]() func() {
	return func() {
		localZero[[]A]()
		localZero[[0]B]()
		localZero[func(A) B]()
	}
}

func (localFactory[T]) receiver() func() {
	return func() { localZero[[]T]() }
}

func Use() {
	localSafe[model.Event, model.Event]()()
	structuralprojection.Safe[model.Event, model.Event]()()
	localFactory[model.Event]{}.receiver()()
	structuralprojection.Factory[model.Event]{}.Receiver()()
	structuralprojection.Safe[model.Event, int]()()
	localSafe[model.Event, int]()()
	structuralprojection.Pick[model.Event, int]()()
	structuralprojection.PickReceiver[model.Event, int]()()
}
