package main

import (
	"testing"

	"example.com/program/library"
)

func testRead() (int, error) {
	var zero_tgo5 int
	var value_tgo3 int
	var error_tgo4 error
	value_tgo3, error_tgo4 = library.Read()
	if error_tgo4 != nil {
		return zero_tgo5, error_tgo4
	}
	return value_tgo3, nil
}

func TestRead(t *testing.T) {
	v, err := testRead()
	if v != 3 || err != nil {
		t.Fatal(v, err)
	}
}
