package main

import fmt_tgo13 "fmt"

func other() error {
	var error_tgo12 error = fail()
	if error_tgo12 != nil {
		return fmt_tgo13.Errorf("fail: %w", error_tgo12)
	}
	return nil
}
