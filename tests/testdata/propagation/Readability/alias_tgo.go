package main

import format "fmt"

func aliased() error {
	format.Print("A")
	var error_tgo1 *failure
	_, _, error_tgo1 = target(true)
	if error_tgo1 != nil {
		return format.Errorf("target: %w", error_tgo1)
	}
	return nil
}
