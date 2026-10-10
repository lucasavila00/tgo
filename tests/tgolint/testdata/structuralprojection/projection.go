package structuralprojection

type Slot[T any] struct{ Value T }

type Factory[T any] struct{}

func zero[T any]() {
	var value T
	_ = value
}

func Compound[A, B any]() func() {
	return func() {
		zero[[]A]()
		zero[[2]B]()
		zero[map[string]*B]()
		zero[chan A]()
		zero[struct {
			First  A
			Second B
		}]()
		zero[func(A) B]()
		zero[Slot[A]]()
	}
}

func Primitive[T any]() func() {
	return func() { zero[T]() }
}

func (Factory[T]) Receiver() func() {
	return func() { zero[[]T]() }
}
