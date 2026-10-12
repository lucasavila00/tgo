package main

import fmt_tgo17 "fmt"

func other() error {
	{
		var tgo16 error = fail()
		if tgo16 != nil {
			return fmt_tgo17.Errorf("fail: %w", tgo16)
		}
	}
	return nil
}
