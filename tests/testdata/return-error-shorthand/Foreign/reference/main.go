package main

import (
	"example.com/shortforeign/foreign"
	"fmt"
)

func values() (any, any, error) {
	var left, right any
	left, right = foreign.Pair()
	return left, right, nil
}

func main() {
	left, right, err := values()
	fmt.Println(left, right, err, foreign.Calls)
}
