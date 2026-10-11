//line virtual.tgo:1
package main

import (
	"errors"
	"fmt"
)

var failure = errors.New("bad")

func fail() error { return failure }

func direct() (tgo2 error) {
	var tgo3 error = tgo2
	{
		var tgo1 error = fail()
		if tgo1 != nil {
			tgo3 = tgo1
			return tgo3
		}
	}
	fmt.Println("continued direct")
	return nil
}
