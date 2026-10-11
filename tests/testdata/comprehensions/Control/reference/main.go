package main

import (
	"errors"
	"fmt"
)

var failure = errors.New("failed")

func read(value int) (int, error) {
	if value == 2 {
		return 0, failure
	}
	return value, nil
}
func propagate(wrapped bool) ([]int, error) {
	defer fmt.Println("deferred")
	values := []int{}
	for x := range 3 {
		v, e := read(x)
		if e != nil {
			if wrapped {
				return nil, fmt.Errorf("read: %w", e)
			}
			return nil, e
		}
		values = append(values, v)
	}
	return values, nil
}
func returnValue() int {
	values := []int{}
	for x := range 3 {
		if x == 1 {
			return 9
		}
		values = append(values, x)
	}
	return len(values)
}
func main() {
	for _, wrapped := range []bool{false, true} {
		values, err := propagate(wrapped)
		fmt.Println(values, err, errors.Is(err, failure), err == failure)
	}
	fmt.Println(returnValue())
	skipped := []int{}
	for x := range 3 {
		if x == 1 {
			continue
		}
		skipped = append(skipped, x)
	}
	fmt.Println(skipped)
	filtered := []int{}
	for x := range 3 {
		if x == 1 {
			continue
		}
		filtered = append(filtered, x)
	}
	fmt.Println(filtered)
	nested := [][2]int{}
	for x := range 2 {
		for y := range 3 {
			if y == 1 {
				break
			}
			nested = append(nested, [2]int{x, y})
		}
	}
	fmt.Println(nested)
	for i := range 2 {
		if i == 0 {
			continue
		}
		values := []int{}
		for x := range 2 {
			values = append(values, x)
		}
		fmt.Println("outer", i, values)
	}
outer:
	for i := range 2 {
		values := []int{}
		for x := range 3 {
			if x == 1 {
				break outer
			}
			values = append(values, x)
		}
		fmt.Println("unreached", i)
	}
	var result []int
	{
		values := []int{}
		for x := range 3 {
			if x == 1 {
				goto done
			}
			values = append(values, x)
		}
		result = values
	}
	fmt.Println("unreached")
done:
	fmt.Println(result)
}
