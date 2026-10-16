package types

import (
	"go/token"
	"testing"
)

// TestEnumTagWidth checks both sides of each integer-width boundary.
func TestEnumTagWidth(t *testing.T) {
	for _, test := range []struct {
		variants int
		want     BasicKind
	}{
		{1, Uint8}, {256, Uint8}, {257, Uint16},
		{65536, Uint16}, {65537, Uint32},
	} {
		if got := enumTagKind(test.variants); got != test.want {
			t.Errorf("%d variants: got %s, want %s", test.variants, Typ[got], Typ[test.want])
		}
	}
}

// TestEnumStorageSizes checks padding and the cost of the shared interface field.
func TestEnumStorageSizes(t *testing.T) {
	sizes := SizesFor("gc", "amd64")
	for _, test := range []struct {
		name  string
		bytes int64
		boxed bool
		want  int64
	}{
		{"inline79", 79, false, 80},
		{"inline80", 80, false, 81},
		{"boxed80", 80, true, 24},
	} {
		t.Run(test.name, func(t *testing.T) {
			enum := enumLayoutFixture(NewArray(Typ[Byte], test.bytes))
			enum.variants[0].boxed = test.boxed
			if got := sizes.Sizeof(enumStorage(enum)); got != test.want {
				t.Errorf("storage size = %d, want %d", got, test.want)
			}
		})
	}
}

// TestEnumStorageComparability keeps slice payloads out of Go comparisons after boxing.
func TestEnumStorageComparability(t *testing.T) {
	enum := enumLayoutFixture(NewSlice(Typ[Int]))
	enum.variants[0].boxed = true
	if Comparable(enumStorage(enum)) {
		t.Fatal("boxing made a slice payload comparable")
	}
	enum = enumLayoutFixture(Typ[Int])
	enum.variants[0].boxed = true
	if !Comparable(enumStorage(enum)) {
		t.Fatal("boxing made an integer payload noncomparable")
	}
}

// TestEnumLargestPayload keeps the first declared payload when two sizes are equal.
func TestEnumLargestPayload(t *testing.T) {
	enum := enumLayoutFixture(Typ[Int64], Typ[Float64], Typ[Int32])
	sizes := SizesFor("gc", "amd64")
	if got := largestEnumPayload(enum, sizes, 0); got != enum.variants[0] {
		t.Fatal("equal sizes did not select the first variant")
	}
	enum.variants[0].boxed = true
	if got := largestEnumPayload(enum, sizes, 0); got != enum.variants[1] {
		t.Fatal("boxed payload was selected again")
	}
}

// enumLayoutFixture creates independent payload types for layout checks.
func enumLayoutFixture(fields ...Type) *Enum {
	pkg := NewPackage("example.test/layout", "layout")
	owner := NewNamed(NewTypeName(token.NoPos, pkg, "Choice", nil), nil, nil)
	tag := NewNamed(NewTypeName(token.NoPos, pkg, "ChoiceTag", nil), Typ[Uint8], nil)
	enum := &Enum{named: owner, tag: tag}
	owner.enum = enum
	for index, field := range fields {
		structure := NewStruct([]*Var{NewField(token.NoPos, pkg, "Value", field, false)}, nil)
		payload := NewNamed(NewTypeName(token.NoPos, pkg, "Payload", nil), structure, nil)
		enum.variants = append(enum.variants, &EnumVariant{enum: enum, typ: payload, index: index})
	}
	return enum
}
