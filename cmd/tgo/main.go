// Command tgo compiles tgo packages to Go source.
package main

import (
	"fmt"
	"os"

	"tgo/internal/driver"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "build" {
		fmt.Fprintln(os.Stderr, "usage: tgo build [./path | ./...]")
		os.Exit(2)
	}
	directory, err := os.Getwd()
	if err == nil {
		err = driver.Build(directory, os.Args[2:])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
