package main

import (
	"testing"

	"example.com/program/library"
)

func testRead() (int, error) {
	var tgo5 int
	{
		var tgo3 int
		var tgo4 error
		tgo3, tgo4 = library.Read()
		if tgo4 != nil {
			return tgo5, tgo4
		}
		return tgo3, nil
	}
}

func TestRead(t *testing.T) {
	v, err := testRead()
	if v != 3 || err != nil {
		t.Fatal(v, err)
	}
}
