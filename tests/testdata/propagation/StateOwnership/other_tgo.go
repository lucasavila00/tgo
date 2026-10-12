package main

import fmt_tgo11 "fmt"

func other() error {
	{
		var tgo10 error = fail()
		if tgo10 != nil {
			return fmt_tgo11.Errorf("fail: %w", tgo10)
		}
	}
	return nil
}
