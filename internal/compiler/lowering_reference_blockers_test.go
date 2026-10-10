package compiler

import "testing"

func TestLoweringRepairsMultipleBlockedTypeReferences(t *testing.T) {
	compileSourceOutput(t, `package sample

type T int
type U int

func factory() struct {
	A T
	B U
} {
	return struct {
		A T
		B U
	}{A: 1, B: 2}
}

func consume(value struct {
	A T
	B U
}, number int) int {
	return int(value.A) + int(value.B) + number
}

func load() (int, error) {
	return 3, nil
}

func use(T int, U int) (int, error) {
	return consume(factory(), load()!!) + T + U, nil
}
`)
}
