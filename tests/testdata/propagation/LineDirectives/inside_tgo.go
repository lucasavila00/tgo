package main

import "fmt"

//line inside.tgo:40
func inside() (tgo5 error) {
	var tgo6 error = tgo5
	{
		var tgo4 error = fail()
		if tgo4 != nil {
			tgo6 = tgo4
			return tgo6
		}
	}
	fmt.Println("continued inside")
	return nil
}
