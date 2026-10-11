package main

import "fmt"

//line inside.tgo:40
func inside() error {
	{
		var tgo2 error = fail()
		if tgo2 != nil {
			return tgo2
		}
	}
	fmt.Println("continued inside")
	return nil
}
