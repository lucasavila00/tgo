package main

import "fmt"

func choose(active bool) string {
	var tgo1 string
	if active {
		fmt.Print("active;")
		name := "yes"
		tgo1 = name
	} else {
		fmt.Print("inactive;")
		tgo1 = "no"
	}
	return tgo1

}

func main() {
	fmt.Println(choose(true))
	fmt.Println(choose(false))
}
