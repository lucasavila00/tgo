package main

import (
	"example.com/shortforeign/foreign"
	"fmt"
)

func values() (any, any, error) {
	{
		var tgo1 any
		var tgo2 any
		tgo1, tgo2 = foreign.Pair()
		return tgo1, tgo2, nil
	}
}

func main() {
	left, right, err := values()
	fmt.Println(left, right, err, foreign.Calls)
}
