package main

import (
	"testing"

	"example.com/program/library"
)

func testRead() (tgo8 int, tgo10 error) {
	tgo9 := tgo8
	tgo11 := tgo10
	tgo12 := tgo11
	{
		tgo6, tgo7 := library.Read()
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
