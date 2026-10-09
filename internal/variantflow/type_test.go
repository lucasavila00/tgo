package variantflow

import "testing"

func TestTypeLattice(t *testing.T) {
	t.Parallel()
	one := Variant(1)
	two := Variant(2)
	three := Variant(130)
	all := All(130)

	if got := Union(one, two); got != Union(two, one) {
		t.Fatal("union is not commutative")
	}
	if got := Intersect(all, three); got != three {
		t.Fatalf("all intersect variant = %v, want %v", got, three)
	}
	if got := Without(Union(one, two), one); got != two {
		t.Fatalf("union without variant = %v, want %v", got, two)
	}
	if tag, ok := Singleton(three); !ok || tag != 130 {
		t.Fatalf("singleton = %d, %t; want 130, true", tag, ok)
	}
	if _, ok := Singleton(Union(one, two)); ok {
		t.Fatal("union is a singleton")
	}
	if !IsNever(Intersect(one, two)) {
		t.Fatal("disjoint intersection is possible")
	}
}
