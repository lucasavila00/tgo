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
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = library.Read()
	if error_tgo2 != nil {
		return error_tgo2
	}
	v := value_tgo1
	fmt.Print(v, message)
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
