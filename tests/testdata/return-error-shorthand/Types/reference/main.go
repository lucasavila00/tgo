package main

import "fmt"

type Data struct{ N int }
type Num uint64

type Failure struct{}

func (*Failure) Error() string { return "failed" }

type StructError struct{}

func (StructError) Error() string { return "struct" }

type PointerError interface {
	error
	~*Failure
}

func zeros(err error) (int, string, [2]int, Data, *int, []int, map[string]int, func() int, chan int, any, error) {
	return 0, "", [2]int{}, Data{}, nil, nil, nil, nil, nil, nil, err
}

func generic[T any](err error) (T, error) {
	var value T
	return value, err
}

func genericError[E PointerError]() (int, E) {
	return 3, nil
}

func concrete(err StructError) (int, StructError) {
	return 0, err
}

func typedNil(err *Failure) (int, error) {
	return 0, err
}

func pointerSuccess() (int, *Failure) {
	return 4, nil
}

func shift(n uint) (Num, error) {
	return 1 << n, nil
}

func conditional(ready bool) (uint8, error) {
	if ready {
		return 127, nil
	}
	return 0, nil
}

func shiftBeforeBranch(n uint, ready bool) (Num, int, error) {
	var left Num = 1 << n
	if ready {
		return left, 1, nil
	}
	return left, 0, nil
}

func nilValue() (any, error) {
	return nil, nil
}

func richSuccess() (int, interface{ error }) {
	return 5, nil
}

func main() {
	failure := &Failure{}
	number, text, array, data, pointer, list, mapping, callback, channel, object, err := zeros(failure)
	fmt.Println(number, text == "", array, data, pointer == nil, list == nil, mapping == nil, callback == nil, channel == nil, object == nil, err == failure)
	value, err := generic[Data](nil)
	fmt.Println(value, err)
	values, err := generic[[]int](nil)
	fmt.Println(values == nil, err)
	number, concreteErr := concrete(StructError{})
	fmt.Println(number, concreteErr.Error())
	number, err = typedNil(nil)
	fmt.Println(number, err == nil)
	number, pointerErr := pointerSuccess()
	fmt.Println(number, pointerErr == nil)
	number, pointerErr = genericError[*Failure]()
	fmt.Println(number, pointerErr == nil)
	shifted, err := shift(63)
	fmt.Println(shifted, err)
	small, err := conditional(true)
	fmt.Println(small, err)
	shifted, smallInt, err := shiftBeforeBranch(63, true)
	fmt.Println(shifted, smallInt, err)
	flag, err := generic[bool](nil)
	fmt.Println(flag, err)
	decimal, err := generic[float64](nil)
	fmt.Println(decimal, err)
	complexValue, err := generic[complex128](nil)
	fmt.Println(complexValue, err)
	object, err = nilValue()
	fmt.Println(object == nil, err)
	number, err = richSuccess()
	fmt.Println(number, err)
}
