//line virtual.tgo:1
package main

import tgo4 "fmt"

import "fmt"

func wrapped() error {
	{
		var tgo3 error = fail()
		if tgo3 != nil {
			return tgo4.Errorf("%s: %w", "fail", tgo3)
		}
	}
	fmt.Println("continued wrapped")
	return nil
}
