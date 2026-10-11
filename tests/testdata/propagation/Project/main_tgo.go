package main

import (
	"embed"
	"fmt"

	"example.com/program/library"
)

//go:embed message.txt
var message string
var _ embed.FS

func run() error {
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = library.Read()
	if tgo2 != nil {
		return tgo2
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
