//line virtual.tgo:1
package main

import fmt_tgo4 "fmt"

import "fmt"

func wrapped() error {
	{
		var tgo3 error = fail()
		if tgo3 != nil {
			return fmt_tgo4.Errorf("fail: %w", tgo3)
		}
	}
	fmt.Println("continued wrapped")
	return nil
}
