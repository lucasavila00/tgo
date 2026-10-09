package tgolint

import (
	"go/types"
	"math/rand"
	"testing"
	"testing/quick"
)

func TestNilTypeLatticeProperties(t *testing.T) {
	property := func(leftByte, middleByte, rightByte uint8) bool {
		left := nilTypeFromMembers(leftByte)
		middle := nilTypeFromMembers(middleByte)
		right := nilTypeFromMembers(rightByte)
		return equalNilType(unionNilTypes(left, middle), unionNilTypes(middle, left)) &&
			equalNilType(
				unionNilTypes(unionNilTypes(left, middle), right),
				unionNilTypes(left, unionNilTypes(middle, right)),
			) &&
			equalNilType(intersectNilTypes(left, middle), intersectNilTypes(middle, left)) &&
			equalNilType(
				intersectNilTypes(intersectNilTypes(left, middle), right),
				intersectNilTypes(left, intersectNilTypes(middle, right)),
			) &&
			equalNilType(unionNilTypes(left, left), left) &&
			equalNilType(intersectNilTypes(left, left), left) &&
			equalNilType(
				intersectNilTypes(left, unionNilTypes(left, middle)), left,
			) &&
			equalNilType(
				unionNilTypes(left, intersectNilTypes(left, middle)), left,
			) &&
			equalNilType(unionNilTypes(left, neverNilType()), left) &&
			equalNilType(intersectNilTypes(left, optionalNilType()), left) &&
			isOptionalNilType(unionNilTypes(left, optionalNilType())) &&
			isNeverNilType(intersectNilTypes(left, neverNilType()))
	}
	configuration := &quick.Config{
		MaxCount: 1_000,
		Rand:     rand.New(rand.NewSource(1)), //nolint:gosec // Tests need stable data.
	}
	if err := quick.Check(property, configuration); err != nil {
		t.Fatal(err)
	}
}

func TestDeclaredNilTypeSeparatesStringAndOptionalString(t *testing.T) {
	stringType := types.Typ[types.String]
	if !isNonNilType(declaredNilType(stringType)) {
		t.Fatal("string must exclude nil")
	}
	optionalString := types.NewPointer(stringType)
	if !isOptionalNilType(declaredNilType(optionalString)) {
		t.Fatal("*string must contain string and nil")
	}
	narrowed := intersectNilTypes(declaredNilType(optionalString), nonNilType())
	if !isNonNilType(narrowed) {
		t.Fatal("a nil check must narrow *string to string")
	}
}

func TestNilBooleanReachability(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		unsafe bool
	}{
		{
			name: "contradictory branch",
			body: "if value != nil && value == nil { need(value) }",
		},
		{
			name: "saved false guard",
			body: "value = nil\nchecked := value != nil\n" +
				"value = nil\nif checked { need(value) }",
		},
		{
			name:   "unsafe branch",
			body:   "if value == nil { need(value) }",
			unsafe: true,
		},
		{
			name: "invalidated true guard",
			body: "value = &Item{}\nchecked := value != nil\n" +
				"value = nil\nif checked { need(value) }",
			unsafe: true,
		},
	}
	for _, test := range tests {
		diagnostics, err := runNilAnalysis("value *Item", test.body)
		if err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if got := len(diagnostics) != 0; got != test.unsafe {
			t.Fatalf(
				"%s: got %d diagnostics; want unsafe=%t",
				test.name, len(diagnostics), test.unsafe,
			)
		}
	}
}
