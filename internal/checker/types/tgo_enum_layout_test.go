package types

import (
	"go/token"
	"slices"
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
			enum := enumLayoutFixture(nil, NewArray(Typ[Byte], test.bytes))
			enum.variants[1].boxed = test.boxed
			if got := sizes.Sizeof(enumStorage(enum)); got != test.want {
				t.Errorf("storage size = %d, want %d", got, test.want)
			}
		})
	}
}

// TestEnumStorageComparability keeps slice payloads out of Go comparisons after boxing.
func TestEnumStorageComparability(t *testing.T) {
	enum := enumLayoutFixture(nil, NewSlice(Typ[Int]))
	enum.variants[1].boxed = true
	if Comparable(enumStorage(enum)) {
		t.Fatal("boxing made a slice payload comparable")
	}
	enum = enumLayoutFixture(nil, Typ[Int])
	enum.variants[1].boxed = true
	if !Comparable(enumStorage(enum)) {
		t.Fatal("boxing made an integer payload noncomparable")
	}
}

// TestEnumLargestPayload keeps the first declared payload when two sizes are equal.
func TestEnumLargestPayload(t *testing.T) {
	enum := enumLayoutFixture(nil, Typ[Int64], Typ[Float64], Typ[Int32])
	sizes := SizesFor("gc", "amd64")
	if got := largestEnumPayload(enum, sizes); got != enum.variants[1] {
		t.Fatal("equal sizes did not select the first variant")
	}
	enum.variants[1].boxed = true
	if got := largestEnumPayload(enum, sizes); got != enum.variants[2] {
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
		var payloadFields []*Var
		if field != nil {
			payloadFields = []*Var{NewField(token.NoPos, pkg, "Value", field, false)}
		}
		structure := NewStruct(payloadFields, nil)
		payload := NewNamed(NewTypeName(token.NoPos, pkg, "Payload", nil), structure, nil)
		enum.variants = append(enum.variants, &EnumVariant{enum: enum, typ: payload, index: index})
	}
	return enum
}

// TestEnumLayoutPolicy checks the complete amd64 layout and its selected boxed payloads.
func TestEnumLayoutPolicy(t *testing.T) {
	bytes := func(size int64) Type { return NewArray(Typ[Byte], size) }
	for _, test := range []struct {
		name   string
		fields []Type
		size   int64
		boxed  []int
	}{
		{"large_first", []Type{bytes(128)}, 129, nil},
		{"large_later", []Type{nil, bytes(128)}, 24, []int{1}},
		{"exact80", []Type{nil, bytes(79)}, 80, nil},
		{"over80", []Type{nil, bytes(80)}, 24, []int{1}},
		{"equal_payloads", []Type{nil, bytes(40), bytes(40)}, 64, []int{1}},
		{"multiple_boxes", []Type{nil, bytes(80), bytes(80)}, 24, []int{1, 2}},
		{"unavoidable_large", []Type{bytes(100), bytes(80), bytes(80)}, 120, []int{1, 2}},
		{"boxing_grows", []Type{bytes(80), Typ[Byte]}, 82, nil},
		{"equal_size_keeps_inline", []Type{bytes(64), bytes(23)}, 88, nil},
		{"later_savings", []Type{bytes(80), bytes(8), bytes(8), bytes(8)}, 104, []int{1, 2, 3}},
	} {
		t.Run(test.name, func(t *testing.T) {
			enum := enumLayoutFixture(test.fields...)
			layout := enumLayout(enum)
			if size := SizesFor("gc", "amd64").Sizeof(layout); size != test.size {
				t.Fatalf("layout size=%d, want %d", size, test.size)
			}
			for index, variant := range enum.variants {
				want := slices.Contains(test.boxed, index)
				if variant.boxed != want {
					t.Errorf("variant %d boxed=%v, want %v", index, variant.boxed, want)
				}
			}
		})
	}
}

// TestEnumBoxedComparisonMarker keeps a large slice payload noncomparable after boxing.
func TestEnumBoxedComparisonMarker(t *testing.T) {
	payload := NewStruct([]*Var{
		NewField(token.NoPos, nil, "Values", NewSlice(Typ[Int]), false),
		NewField(token.NoPos, nil, "Bytes", NewArray(Typ[Byte], 80), false),
	}, nil)
	enum := enumLayoutFixture(nil, payload)
	layout := enumLayout(enum)
	if !enum.variants[1].boxed || Comparable(layout) {
		t.Fatal("large slice payload must be boxed without making the enum comparable")
	}
	if size := SizesFor("gc", "amd64").Sizeof(layout); size != 32 {
		t.Fatalf("layout size=%d, want 32 including comparison marker padding", size)
	}
}
