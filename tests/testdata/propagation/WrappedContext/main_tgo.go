package main

import fmt_tgo2 "fmt"

import (
	"errors"
	"fmt"
)

var failure = errors.New("failure")

func run() error {
	calls := map[string]func() error{
		"%s %w %%": func() error { return failure },
	}
	{
		var tgo1 error = calls["%s %w %%"]()
		if tgo1 != nil {
			return fmt_tgo2.Errorf("calls[\"%%s %%w %%%%\"]: %w", tgo1)
		}
	}
	return nil
}

func main() {
	err := run()
	fmt.Printf("%v %t\n", err, errors.Is(err, failure))
}
