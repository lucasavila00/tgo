package types_test

import (
	"strings"
	"testing"
)

const enumGenericSource = `type Account enum {
 Personal struct { Name string }
 Closed
}
type Raw = struct { tgoTag AccountTag; tgoVariant0 AccountPersonal }
type Other Account
`

// TestEnumGenericStorage checks the declaration and instantiation boundaries of generic code.
func TestEnumGenericStorage(t *testing.T) {
	cases := []struct {
		name, source, diagnostic string
	}{
		{"any_zero", `func zero[T any]() T { var value T; return value }; var x = zero[Account]()`, ""},
		{"exact_zero", `func zero[T Account]() T { return T{} }; var x = zero[Account]()`, ""},
		{"same_family", `func convert[T Account|Other](v Account) T { return T(v) }`, ""},
		{"multi_enum_identity", `type Second enum { Empty }
 func identity[T Account|Second](v T) T { return v }`, ""},
		{"multi_enum_zero", `type Second enum { Empty }
 func zero[T Account|Second]() T { var v T; return v }; var x = zero[Second]()`, ""},
		{"identity", `func identity[T Account|Other](v T) T { return v }`, ""},
		{"intersection", `func zero[T interface { Account; ~Raw }]() T { return T{} }
 var x = zero[Account]()`, ""},
		{"ordinary_raw", `func forge[T ~Raw]() T { return T{tgoTag:99} }; var x = forge[Raw]()`, ""},
		{"dependent_pointer", `func keep[U Account|Other,T ~*U](v T) T { return v }
 func outer[U Account|Other](v *U) *U { return keep[U](v) }`, ""},
		{"slice_container", `func create[T ~[]Account]() T { return make(T,1) }; var x = create[[]Account]()`, ""},
		{"explicit_call", `func forge[T ~Raw]() T { return T{tgoTag:99} }; var x = forge[Account]()`, "constraint"},
		{"inferred_call", `func forge[T ~Raw](v T) T { return T{tgoTag:99} }; var x = forge(Account{})`, "constraint"},
		{"partial_call", `func forge[T ~Raw,U any](v T,u U) T { return T{tgoTag:99} }
 var x = forge[Account](Account{},1)`, "constraint"},
		{"function_value", `func forge[T ~Raw]() T { return T{tgoTag:99} }; var x = forge[Account]`, "constraint"},
		{"inferred_function_value", `func forge[T ~Raw](v T) T { return T{tgoTag:99} }
 var x func(Account) Account = forge`, "constraint"},
		{"generic_type", `type Box[T ~Raw] struct { Value T }; var x Box[Account]`, "constraint"},
		{"generic_alias", `type Box[T ~Raw] = struct { Value T }; var x Box[Account]`, "constraint"},
		{"forwarded_parameter", `func raw[T ~Raw](v T) T { return v }
 func forward[T Account](v T) T { return raw(v) }`, "constraint"},
		{"forwarded_intersection", `func raw[T ~Raw](v T) T { return v }
 func forward[T interface {Account; ~Raw}](v T) T { return raw(v) }`, "constraint"},
		{"pointer_argument", `func raw[T ~*Raw](v T) T { return v }; var x = raw(new(Account))`, "does not satisfy"},
		{"pointer_exact", `func keep[T ~*Account](v T) T { return v }; var x = keep(new(Account))`, ""},
		{"pointer_expose", `func expose[T ~*Account](v T) *Raw { return (*Raw)(v) }`, "storage"},
		{"generic_literal", `func forge[T Account]() T { return T{tgoTag:99} }`, "storage is private"},
		{"union_literal", `func forge[T Account|Other]() T { return T{tgoTag:99} }`, "storage is private"},
		{"intersection_literal", `func forge[T interface {Account; ~Raw}]() T { return T{tgoTag:99} }`, "storage is private"},
		{"conversion_target", `func forge[T Account]() T { return T(Raw{}) }`, "storage"},
		{"conversion_source", `func expose[T Account](v T) Raw { return Raw(v) }`, "storage"},
		{"assignment_target", `func forge[T Account]() T { var v T = Raw{}; return v }`, "storage"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := checkTGo(t, enumGenericSource+test.source)
			if test.diagnostic == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.diagnostic) {
				t.Fatalf("want %q, got %v", test.diagnostic, err)
			}
		})
	}
}
