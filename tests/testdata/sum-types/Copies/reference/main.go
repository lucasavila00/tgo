package main

import "fmt"

type personal struct {
	name    string
	counter *int
}

func main() {
	count := 1
	value := personal{name: "Ada", counter: &count}
	selected := value
	company := "Acme"
	*selected.counter = 9
	saved := func() string { return selected.name }
	fmt.Println(selected.name, *selected.counter)
	fmt.Println(saved(), count, company)
	copy := personal{name: "Grace", counter: &count}
	payload := copy
	payload.name = "changed"
	fmt.Println(copy.name, payload.name)
}
