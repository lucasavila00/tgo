package main

import (
	"errors"
	"fmt"
)

type account struct {
	business bool
	text     string
}

var failure = errors.New("failed")

func read(value string) (int, error) {
	if value == "fail" {
		return 0, failure
	}
	return len(value), nil
}
func number(value account) (int, error) {
	if !value.business && value.text == "return" {
		return -1, nil
	}
	n, err := read(value.text)
	if err != nil {
		if value.business {
			return 0, fmt.Errorf("read: %w", err)
		}
		return 0, err
	}
	return n, nil
}
func nestedReturn(value account, active bool) string {
	if active {
		if value.business {
			return "business:" + value.text
		}
		return "personal:" + value.text
	}
	return "inactive"
}

func jumps() {
	values := []account{{text: "continue"}, {text: "break"}, {business: true, text: "ordinary"}, {business: true, text: "goto"}}
	var trace []string
outer:
	for _, value := range values {
	selected:
		switch value.business {
		case false:
			if value.text == "continue" {
				continue outer
			}
			trace = append(trace, value.text)
			break selected
		case true:
			trace = append(trace, value.text)
			if value.text == "goto" {
				goto done
			}
			break
		}
		trace = append(trace, "tail")
	}
done:
	fmt.Println(trace)
}
func main() {
	fmt.Println(nestedReturn(account{text: "Ada"}, true))
	fmt.Println(nestedReturn(account{business: true, text: "Acme"}, true))
	fmt.Println(nestedReturn(account{text: "skip"}, false))
	jumps()
	value, err := number(account{text: "return"})
	fmt.Println(value, err)
	value, err = number(account{text: "fail"})
	fmt.Println(value, err == failure)
	value, err = number(account{business: true, text: "fail"})
	fmt.Println(value, err != failure, errors.Is(err, failure))
	value, err = number(account{business: true, text: "ok"})
	fmt.Println(value, err)
}
