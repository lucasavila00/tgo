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
	var zero_tgo1 int
	var zero_tgo2 string
	var zero_tgo3 [2]int
	var zero_tgo4 Data
	var zero_tgo5 *int
	var zero_tgo6 []int
	var zero_tgo7 map[string]int
	var zero_tgo8 func() int
	var zero_tgo9 chan int
	var zero_tgo10 any
	return zero_tgo1, zero_tgo2, zero_tgo3, zero_tgo4, zero_tgo5, zero_tgo6, zero_tgo7, zero_tgo8, zero_tgo9, zero_tgo10, err
}

func generic[T any](err error) (T, error) {
	var zero_tgo11 T
	return zero_tgo11, err
}

func genericError[E PointerError]() (int, E) {
	return 3, nil
}

func concrete(err StructError) (int, StructError) {
	var zero_tgo12 int
	return zero_tgo12, err
}

func typedNil(err *Failure) (int, error) {
	var zero_tgo13 int
	return zero_tgo13, err
}

func pointerSuccess() (int, *Failure) {
	return 4, nil
}

func shift(n uint) (Num, error) {
	return 1 << n, nil
}

func conditional(ready bool) (uint8, error) {
	var tgo14 uint8
	if ready {
		tgo14 = 127
	} else {
		tgo14 = 0
	}
	return tgo14, nil
}

func shiftBeforeBranch(n uint, ready bool) (Num, int, error) {
	var value_tgo16 Num = 1 << n
	var tgo15 int
	if ready {
		tgo15 = 1
	} else {
		tgo15 = 0
	}
	return value_tgo16, tgo15, nil
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
