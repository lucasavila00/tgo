package main

import (
	"testing"

	"example.com/program/library"
)

func testRead() (tgo7 int, tgo9 error) {
	var tgo8 int = tgo7
	var tgo10 error = tgo9
	{
		var tgo5 int
		var tgo6 error
		tgo5, tgo6 = library.Read()
		if tgo6 != nil {
			tgo10 = tgo6
			return tgo8, tgo10
		}
		return tgo5, nil
	}
}

func TestRead(t *testing.T) {
	v, err := testRead()
	if v != 3 || err != nil {
		t.Fatal(v, err)
	}
}
