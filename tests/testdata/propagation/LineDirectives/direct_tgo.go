//line virtual.tgo:1
package main

import (
	"errors"
	"fmt"
)

var failure = errors.New("bad")

func fail() error { return failure }

func direct() error {
	var error_tgo1 error = fail()
	if error_tgo1 != nil {
		return error_tgo1
	}
	fmt.Println("continued direct")
	return nil
}
