package main

import (
	"fmt"
	"runtime"
)

type personal struct {
	name    string
	counter *int
}

type archive struct {
	values  [128]byte
	counter *int
}

func makeArchive() *archive {
	count := 11
	return &archive{values: [128]byte{7}, counter: &count}
}

func boxedCopies() {
	original := makeArchive()
	copied := original
	payload := *original
	original = nil
	runtime.GC()
	*payload.counter = 23
	payload.values[0] = 99
	fmt.Println(copied.values[0], payload.values[0], *copied.counter)
	fmt.Println(original == nil)
}

func main() {
	boxedCopies()
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
