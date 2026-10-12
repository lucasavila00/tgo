package main

import (
	"errors"
	"fmt"
)

var events []string
var token = &struct{ N int }{8}

func done() { events = append(events, "D") }

func skipping(err error) (int, error) {
	var zero_tgo2 int
	for i := 0; i < 2; i++ {
		var tgo1 int
		if i == 0 {
			continue
			_ = 0 == 0
		} else {
			defer done()
			tgo1 = 7
		}
		return tgo1, nil
	}
	return zero_tgo2, err
}

func breaking(err error) (int, error) {
	var zero_tgo4 int
	for {
		var tgo3 int
		if true {
			break
			_ = 0 == 0
		} else {
			tgo3 = 1
		}
		return tgo3, nil
	}
	return zero_tgo4, err
}

func jumping(err error) (int, error) {
	var zero_tgo6 int
	var value int
	{
		var tgo5 int
		if true {
			goto exit
			_ = 0 == 0
		} else {
			tgo5 = 1
		}
		value = tgo5
	}
	return value, nil
exit:
	{
		return zero_tgo6, err
	}
}

func onlyError(err error) error {
	return err
}

func ordinaryNamed() (value int, err error) {
	call := func() (int, error) { return 9, nil }
	return call()
}

func panicking() (int, error) {
	defer done()
	var tgo7 int
	if true {
		panic(token)
		_ = 0 == 0
	} else {
		tgo7 = 1
	}
	return tgo7, nil
}

func main() {
	failure := errors.New("failure")
	events = nil
	value, err := skipping(failure)
	fmt.Println(value, err, events)
	value, err = breaking(failure)
	fmt.Println(value, err == failure)
	value, err = jumping(failure)
	fmt.Println(value, err == failure)
	fmt.Println(onlyError(failure) == failure)
	value, err = ordinaryNamed()
	fmt.Println(value, err)
	events = nil
	func() {
		defer func() { fmt.Println(recover() == token, events) }()
		_, _ = panicking()
	}()
}
