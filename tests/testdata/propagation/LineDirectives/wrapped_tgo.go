//line virtual.tgo:1
package main

import "fmt"

func wrapped() error {
	var error_tgo3 error = fail()
	if error_tgo3 != nil {
		return fmt.Errorf("fail: %w", error_tgo3)
	}
	fmt.Println("continued wrapped")
	return nil
}
