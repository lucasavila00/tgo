package main

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
	{
		type T string
		type Local = T
		direct, err := pair(x, T("local"))
		if err != nil {
			return err
		}
		wrapped, err := pair(x, T("local"))
		if err != nil {
			return fmt.Errorf("pair: %w", err)
		}
		aliased, err := pair(x, Local("alias"))
		if err != nil {
			return err
		}
		fmt.Println(direct.A, direct.B, direct.T, *direct.Nested[0][direct.B], direct.G.A, direct.G.B)
		fmt.Println(wrapped.F(wrapped.A, wrapped.B))
		fmt.Println(aliased.G.A, aliased.G.B)
	}
	return nil
}

func main() {
	fmt.Println(run(7))
}
