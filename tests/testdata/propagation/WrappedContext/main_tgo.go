package main

import (
	"errors"
	"fmt"
)

var failure = errors.New("failure")

func run() error {
	calls := map[string]func() error{
		"%s %w %%": func() error { return failure },
	}
	var error_tgo1 error = calls["%s %w %%"]()
	if error_tgo1 != nil {
		return fmt.Errorf("calls[\"%%s %%w %%%%\"]: %w", error_tgo1)
	}
	return nil
}

func main() {
	err := run()
	fmt.Printf("%v %t\n", err, errors.Is(err, failure))
}
