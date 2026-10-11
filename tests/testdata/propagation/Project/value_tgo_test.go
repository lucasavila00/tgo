package main

import (
	"testing"

	"example.com/program/library"
)

func testRead() (tgo8 int, tgo10 error) {
	var tgo9 int = tgo8
	var tgo11 error = tgo10
	var tgo12 error = tgo11
	{
		var tgo6 int
		var tgo7 error
		tgo6, tgo7 = library.Read()
		tgo11 = tgo7
		if tgo11 != tgo12 {
			return tgo9, tgo11
		}
		return tgo6, nil
	}
}

func TestRead(t *testing.T) {
	v, err := testRead()
	if v != 3 || err != nil {
		t.Fatal(v, err)
	}
}
