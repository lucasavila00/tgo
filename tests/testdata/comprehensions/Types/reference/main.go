package main

import "fmt"

type Number int
type Numbers []Number
type Lookup map[int]Number

var initial = func() Numbers {
	r := Numbers{}
	for x := range 3 {
		r = append(r, Number(x))
	}
	return r
}()

func twice[S ~[]E, E ~int](values S) S {
	r := S{}
	for _, x := range values {
		r = append(r, x*2)
	}
	return r
}
func copyMap[M ~map[K]V, K comparable, V any](values M) M {
	r := M{}
	for k, v := range values {
		r[k] = v
	}
	return r
}
func main() {
	xs := Numbers{2, 3}
	fmt.Println(initial, twice(xs), copyMap(Lookup{1: 2}))
	shifts := []uint64{}
	for n := range 2 {
		var v uint64 = 1 << n
		shifts = append(shifts, v)
	}
	fmt.Println(shifts)
	pointers := []*int{}
	for range 2 {
		pointers = append(pointers, nil)
	}
	fmt.Println(pointers)
	interfaces := []any{}
	for _, x := range xs {
		var v any
		if x > 2 {
			v = x
		} else {
			v = "small"
		}
		interfaces = append(interfaces, v)
	}
	fmt.Println(interfaces)
	mapping := map[any]any{}
	for _, x := range xs {
		mapping[x] = x
	}
	fmt.Println(mapping)
	pointed := Numbers{}
	for _, x := range xs {
		pointed = append(pointed, x)
	}
	pointer := &pointed
	fmt.Println(*pointer)
	shadowed := Numbers{}
	for _, Numbers := range xs {
		shadowed = append(shadowed, Numbers)
	}
	fmt.Println(shadowed)
	filtered := []Number{}
	for _, x := range xs {
		if x > 0 {
			filtered = append(filtered, x)
		}
	}
	fmt.Println(filtered, "outer")
	a := []int{}
	for x := range 2 {
		a = append(a, x)
	}
	fmt.Println(a)
	l := []int{}
	for x := range 2 {
		l = append(l, x)
	}
	fmt.Println(l)
	m := []int{}
	for x := range 2 {
		m = append(m, x)
	}
	fmt.Println(m)
	empty := Lookup{}
	for k, v := range (Lookup{}) {
		empty[k] = v
	}
	fmt.Println(empty == nil)
}
