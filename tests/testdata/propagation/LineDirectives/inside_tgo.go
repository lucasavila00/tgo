package main

import "fmt"

//line inside.tgo:40
func inside() error {
	var error_tgo2 error = fail()
	if error_tgo2 != nil {
		return error_tgo2
	}
	fmt.Println("continued inside")
	return nil
}
