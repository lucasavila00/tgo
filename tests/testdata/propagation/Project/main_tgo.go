package main

import (
	"embed"
	"fmt"

	"example.com/program/library"
)

//go:embed message.txt
var message string
var _ embed.FS

func run() (tgo3 error) {
	var tgo4 error = tgo3
	var tgo5 error = tgo4
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = library.Read()
	tgo4 = tgo2
	if tgo4 != tgo5 {
		return tgo4
	}
	v := tgo1
	fmt.Print(v, message)
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
