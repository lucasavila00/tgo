package main

import "fmt"

type tree struct {
	branch   bool
	value    int
	children []tree
	parent   *tree
}

func total(value tree) int {
	if !value.branch {
		return value.value
	}
	sum := 0
	for _, child := range value.children {
		sum += total(child)
	}
	return sum
}
func main() {
	leaf := tree{value: 3}
	value := tree{branch: true, children: []tree{leaf, {value: 4}}, parent: &leaf}
	fmt.Println(total(value), value.parent.value)
	*value.parent = tree{value: 9}
	fmt.Println(total(value), leaf.value)
}
