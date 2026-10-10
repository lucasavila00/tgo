package structuralprojection

type Slot[T any] struct{ Value T }

type Factory[T any] struct{}

func zero[T any]() {
	var value T
	_ = value
}

func Safe[A, B any]() func() {
	return func() {
		zero[[]A]()
		zero[[0]B]()
		zero[map[string]*B]()
		zero[chan A]()
		zero[func(A) B]()
	}
}

func Invalid[A, B any]() func() {
	return func() {
		zero[[2]A]()
		zero[struct{ Value B }]()
		zero[Slot[A]]()
	}
}

func Primitive[T any]() func() {
	return func() { zero[T]() }
}

func (Factory[T]) Receiver() func() {
	return func() { zero[[]T]() }
}

func (Factory[T]) ReceiverPrimitive() func() {
	return func() { zero[T]() }
}

func Pick[A, B any]() func() {
	factory := Primitive[B]
	return factory()
}

func PickReceiver[A, B any]() func() {
	factory := Factory[B]{}.ReceiverPrimitive
	return factory()
}
