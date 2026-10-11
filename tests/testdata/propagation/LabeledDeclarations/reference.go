package main

import "fmt"

func reference(fail, wrap bool, visits int) error {
	defer func() { trace = append(trace, "defer") }()
	for i := 0; i < visits; i++ {
		value, err := next(fail)
		if err != nil {
			if wrap {
				return fmt.Errorf("next: %w", err)
			}
			return err
		}
		trace = append(trace, fmt.Sprint(value))
	}
	return nil
}
