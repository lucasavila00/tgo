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

func localCompound[A, B any]() func() {
	return func() {
		localZero[[]A]()
		localZero[[2]B]()
		localZero[struct {
			First  A
			Second B
		}]()
		localZero[func(A) B]()
	}
}

func (localFactory[T]) receiver() func() {
	return func() { localZero[[]T]() }
}

func Use() {
	localCompound[model.Event, model.Event]()()
	structuralprojection.Compound[model.Event, model.Event]()()
	localFactory[model.Event]{}.receiver()()
	structuralprojection.Factory[model.Event]{}.Receiver()()
	structuralprojection.Compound[model.Event, int]()()
	localCompound[model.Event, int]()()
}
