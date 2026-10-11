package main

import tgo8 "fmt"

import "fmt"

type box[A, B any] struct {
	A A
	B B
}

func pair[A any, B comparable](a A, b B) (struct {
	A      A
	B      B
	T      B
	Nested []map[B]*A
	Array  [1]*B
	C      <-chan B
	F      func(A, ...B) (A, B)
	G      box[A, B]
	I      interface{ Call(A, ...B) B }
}, error) {
	return struct {
		A      A
		B      B
		T      B
		Nested []map[B]*A
		Array  [1]*B
		C      <-chan B
		F      func(A, ...B) (A, B)
		G      box[A, B]
		I      interface{ Call(A, ...B) B }
	}{
		A: a, B: b, T: b,
		Nested: []map[B]*A{{b: &a}},
		Array:  [1]*B{&b},
		F:      func(a A, b ...B) (A, B) { return a, b[0] },
		G:      box[A, B]{a, b},
	}, nil
}

func run[T any](x T) (tgo4 error) {
	type tgo3 = T
	var tgo5 error = tgo4

	{
		type T string
		type Local = T
		var tgo1 struct {
			A      tgo3
			B      Local
			T      Local
			Nested []map[Local]*tgo3
			Array  [1]*Local
			C      <-chan Local
			F      func(tgo3, ...Local) (tgo3, Local)
			G      box[tgo3, Local]
			I      interface {
				Call(tgo3, ...Local) Local
			}
		}
		var tgo2 error
		tgo1, tgo2 = pair(x, Local("local"))
		if tgo2 != nil {
			tgo5 = tgo2
			return tgo5
		}
		direct := tgo1
		var tgo6 struct {
			A      tgo3
			B      Local
			T      Local
			Nested []map[Local]*tgo3
			Array  [1]*Local
			C      <-chan Local
			F      func(tgo3, ...Local) (tgo3, Local)
			G      box[tgo3, Local]
			I      interface {
				Call(tgo3, ...Local) Local
			}
		}
		var tgo7 error
		tgo6, tgo7 = pair(x, Local("local"))
		if tgo7 != nil {
			tgo5 = tgo7
			tgo5 = tgo8.Errorf("%s: %w", "pair", tgo5)
			return tgo5
		}
		wrapped := tgo6
		fmt.Println(direct.A, direct.B, direct.T, *direct.Nested[0][direct.B], direct.G.A, direct.G.B)
		fmt.Println(wrapped.F(wrapped.A, wrapped.B))
	}
	return nil
}

func main() {
	fmt.Println(run(7))
}
