package main

import (
	"errors"
	"fmt"
)

var events []string
var token = &struct{ N int }{8}

func done() { events = append(events, "D") }

func skipping(err error) (int, error) {
	for i := 0; i < 2; i++ {
		if i == 0 {
			continue
		}
		defer done()
		return 7, nil
	}
	return 0, err
}

func breaking(err error) (int, error) {
	for {
		break
	}
	return 0, err
}

func jumping(err error) (int, error) {
	var value int
	if true {
		goto exit
	}
	value = 1
	return value, nil
exit:
	return 0, err
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
	panic(token)
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
