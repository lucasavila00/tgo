package main

import "fmt"

type personal struct{ name string }

func fromDependency(name string) personal { return personal{name: name} }
func main() {
	value := fromDependency("dependency")
	fmt.Println(value.name)
	fmt.Println("local")
}
