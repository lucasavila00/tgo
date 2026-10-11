package main

import "fmt"

type record struct {
	value any
}

func identity[T any](value T) T {
	return value
}

func checkPanic(value any) {
	defer func() {
		if recover() != value {
			panic("wrong panic value")
		}
	}()
	panic(value)
}

func remove[M ~map[*int]int](mapping M) {
	delete(mapping, nil)
}

func main() {
	checkPanic(30)
	checkPanic("other")
	mappingWithNil := map[*int]int{nil: 30}
	remove(mappingWithNil)
	mappingWithNil[nil] = 31
	remove(mappingWithNil)
	fmt.Println("delete", len(mappingWithNil))
	ok := true
	var text any
	var keyed, positional record
	var key string
	var mapped, element any
	var first int
	var pointer *record
	if ok {
		text, keyed, positional = "text", record{1}, record{"three"}
		key, mapped, element, first = "key", 5, 7, 9
		pointer = &record{12}
	} else {
		text, keyed, positional = 1, record{"two"}, record{4}
		key, mapped, element, first = "other", "six", "eight", 10
		pointer = &record{13}
	}
	mapping := map[string]any{key: mapped}
	values := []any{element}
	array := [2]int{first, 11}
	fmt.Println(text, keyed.value, positional.value, mapping["key"], values[0], array, pointer.value)

	numbers := []int{10, 20, 30}
	index, replacement, increment := 0, 22, 0
	if ok {
		index, replacement, increment = 1, 21, 2
	}
	numbers[index] = replacement
	numbers[increment]++
	unary, high := 4, 2
	if ok {
		unary, high = 3, 3
	}
	fmt.Println(-unary, numbers[index], numbers[index:high])
	generic, converted := 15, int8(17)
	if ok {
		generic, converted = 14, 16
	}
	fmt.Println(identity(generic), converted)
	call := func(value int) int { return value + 2 }
	argument, asserted := 19, any(1)
	if ok {
		call = func(value int) int { return value + 1 }
		argument, asserted = 18, text
	}
	fmt.Println(call(argument), asserted.(string))

	value := 0
	if ok {
		value = 20
	}
	condition := false
	if ok {
		condition = value > 0
	}
	if condition {
		fmt.Println("if", value)
	}
	i := 2
	if ok {
		i = 0
	}
	for {
		condition = false
		if ok {
			condition = i < 2
		}
		if !condition {
			break
		}
		fmt.Println("for", i)
		step := 2
		if ok {
			step = 1
		}
		i += step
	}
	ranged := []int{}
	if ok {
		ranged = []int{22, 23}
	}
	for _, value := range ranged {
		fmt.Println("range", value)
	}
	value, tag, caseValue := 0, 0, 25
	if ok {
		value, tag, caseValue = 24, 24, 24
	}
	switch tag {
	case caseValue:
		fmt.Println("switch", value)
	default:
		panic("wrong case")
	}
	asserted = any(1)
	if ok {
		asserted = text
	}
	switch value := asserted.(type) {
	case string:
		fmt.Println("type", value)
	default:
		panic("wrong type")
	}

	channel := make(chan any, 1)
	var sent any = "other"
	if ok {
		sent = 26
	}
	channel <- sent
	fmt.Println("send", <-channel)
	if ok {
		sent = 27
	}
	select {
	case channel <- sent:
		fmt.Println("select", <-channel)
	default:
		panic("blocked send")
	}
	if ok {
		sent = 28
	}
	go func(value any) { channel <- value }(sent)
	fmt.Println("go", <-channel)
	if ok {
		sent = 29
	}
	defer fmt.Println("defer", sent)
}
