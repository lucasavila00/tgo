package main

import "fmt"

var calls int
var first = next()
var selected = first
var callback = func() string { return "ready" }

func next() int { calls++; return 7 }
func main()     { fmt.Println(selected, callback(), calls) }
