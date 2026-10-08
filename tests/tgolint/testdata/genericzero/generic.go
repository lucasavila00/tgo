package genericzero

import "example.com/tgolint/model"

type Slot[T any] struct {
	Value T
}

type Factory[T any] struct{}

func (Factory[T]) Variable() {
	var value T
	_ = value
}

func (Factory[T]) Make(length int) []T {
	return make([]T, length)
}

func Variable[T any]() {
	var value T
	_ = value
}

func Named[T any]() (value T) {
	return
}

func New[T any]() *T {
	return new(T)
}

func Make[T any](length int) []T {
	return make([]T, length)
}

func Maybe[T any](enabled bool) {
	if enabled {
		var value T
		_ = value
	}
}

func Mutated[T any](enabled bool) {
	enabled = false
	if enabled {
		var value T
		_ = value
	}
}

func Unless[T any](skip bool) {
	if skip {
		return
	}
	var value T
	_ = value
}

func Recursive[T any](enabled bool) {
	if enabled {
		var value T
		_ = value
		Recursive[T](enabled)
	}
}

func Never[T any]() {
	if false {
		var value T
		_ = value
	}
}

func Dead[T any]() {
	return
	var value T
	_ = value
}

func Nested[T any]() func() {
	return func() {
		var value T
		_ = value
	}
}

func Clear[T any](values []T) {
	clear(values)
}

func MapRead[K comparable, V any](values map[K]V, key K) V {
	return values[key]
}

func ChannelRead[T any](values <-chan T) T {
	return <-values
}

func Assert[T any](value any) T {
	return value.(T)
}

func MapChecked[K comparable, V any](values map[K]V, key K) bool {
	value, ok := values[key]
	if !ok {
		return false
	}
	_ = value
	return true
}

func ChannelChecked[T any](values <-chan T) bool {
	value, ok := <-values
	if !ok {
		return false
	}
	_ = value
	return true
}

func AssertChecked[T any](input any) bool {
	value, ok := input.(T)
	if !ok {
		return false
	}
	_ = value
	return true
}

func MapDiscarded[K comparable, V any](values map[K]V, key K) V {
	value, _ := values[key]
	return value
}

func ChannelDiscarded[T any](values <-chan T) T {
	value, _ := <-values
	return value
}

func AssertDiscarded[T any](input any) T {
	value, _ := input.(T)
	return value
}

func Reslice[T any](values []T, length int) []T {
	return values[:length]
}

func OmittedField[T any]() Slot[T] {
	return Slot[T]{}
}

func OmittedArray[T any]() [1]T {
	return [1]T{}
}

func OmittedSlice[T any](value T) []T {
	return []T{1: value}
}

type eventLike interface {
	TgoTag() uint8
	TgoStarted() model.EventStarted
}

type Reader[T eventLike] struct{}

func (Reader[T]) Started(event T) string {
	return event.TgoStarted().ID
}

func Started[T eventLike](event T) string {
	return event.TgoStarted().ID
}

func MaybeStarted[T eventLike](enabled bool, event T) string {
	if enabled {
		return event.TgoStarted().ID
	}
	return ""
}

func UnlessStarted[T eventLike](skip bool, event T) string {
	if skip {
		return ""
	}
	return event.TgoStarted().ID
}

func EmptySlice[T any]() []T {
	return make([]T, 0)
}

func CurrentSlice[T any](values []T) []T {
	return values[:len(values)]
}

func FullField[T any](value T) Slot[T] {
	return Slot[T]{Value: value}
}

func FullArray[T any](value T) [1]T {
	return [1]T{value}
}
