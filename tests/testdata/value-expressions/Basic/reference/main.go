package main

import "fmt"

func choose(active bool) string {
	if active {
		fmt.Print("active;")
		name := "yes"
		return name
	} else {
		fmt.Print("inactive;")
		return "no"
	}
}

func main() {
	fmt.Println(choose(true))
	fmt.Println(choose(false))
}
