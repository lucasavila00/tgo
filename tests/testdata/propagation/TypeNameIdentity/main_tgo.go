package main

import fmt_tgo6 "fmt"

import "fmt"

type echoed[A any] interface {
	Echo(A) A
}

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
	C      chan (<-chan B)
	F      func(A, ...B) (A, B)
	G      box[A, B]
	I      interface {
		echoed[A]
		Call(A, ...B) B
	}
}, error) {
	return struct {
		A      A
		B      B
		T      B
		Nested []map[B]*A
		Array  [1]*B
		C      chan (<-chan B)
		F      func(A, ...B) (A, B)
		G      box[A, B]
		I      interface {
			echoed[A]
			Call(A, ...B) B
		}
	}{
		A: a, B: b, T: b,
		Nested: []map[B]*A{{b: &a}},
		Array:  [1]*B{&b},
		F:      func(a A, b ...B) (A, B) { return a, b[0] },
		G:      box[A, B]{a, b},
	}, nil
}

func run[T any](x T) error {
	type tgo3 = T
	{
		type T string
		type Local = T
		var tgo1 struct {
			A      tgo3
			B      T
			T      T
			Nested []map[T]*tgo3
			Array  [1]*T
			C      chan (<-chan T)
			F      func(tgo3, ...T) (tgo3, T)
			G      box[tgo3, T]
			I      interface {
				Call(tgo3, ...T) T
				echoed[tgo3]
			}
		}
		var tgo2 error
		tgo1, tgo2 = pair(x, T("local"))
		if tgo2 != nil {
			return tgo2
		}
		direct := tgo1
		var tgo4 struct {
			A      tgo3
			B      T
			T      T
			Nested []map[T]*tgo3
			Array  [1]*T
			C      chan (<-chan T)
			F      func(tgo3, ...T) (tgo3, T)
			G      box[tgo3, T]
			I      interface {
				Call(tgo3, ...T) T
				echoed[tgo3]
			}
		}
		var tgo5 error
		tgo4, tgo5 = pair(x, T("local"))
		if tgo5 != nil {
			return fmt_tgo6.Errorf("pair: %w", tgo5)
		}
		wrapped := tgo4
		var tgo7 struct {
			A      tgo3
			B      Local
			T      Local
			Nested []map[Local]*tgo3
			Array  [1]*Local
			C      chan (<-chan Local)
			F      func(tgo3, ...Local) (tgo3, Local)
			G      box[tgo3, Local]
			I      interface {
				Call(tgo3, ...Local) Local
				echoed[tgo3]
			}
		}
		var tgo8 error
		tgo7, tgo8 = pair(x, Local("alias"))
		if tgo8 != nil {
			return tgo8
		}
		aliased := tgo7
		fmt.Println(direct.A, direct.B, direct.T, *direct.Nested[0][direct.B], direct.G.A, direct.G.B)
		fmt.Println(wrapped.F(wrapped.A, wrapped.B))
		fmt.Println(aliased.G.A, aliased.G.B)
	}
	return nil
}

func main() {
	fmt.Println(run(7))
}
