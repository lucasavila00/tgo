package main

import "fmt"

// reference checks propagation with ordinary Go control flow.
func reference(fail, wrap bool) error {
	defer func() { trace = append(trace, "defer") }()
	value, err := next(fail)
	if err != nil {
		if wrap {
			return fmt.Errorf("next: %w", err)
		}
		return err
	}
	trace = append(trace, fmt.Sprint(value))
	return nil
}
