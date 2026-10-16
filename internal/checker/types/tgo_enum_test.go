package types_test

import (
	"strings"
	"testing"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// TestEnumConstruction checks parent values, copied payload API, and ordinary Go constructors.
func TestEnumConstruction(t *testing.T) {
	info, err := checkTGo(t, `type Account enum {
		Personal struct { Name string }
		Business struct { Company string }
		Closed
	}
	var initial Account
	var made = Account.Personal{Name: "Lucas"}
	var throughGo = NewAccountBusiness("Acme")
	func f() string { return throughGo.BusinessPayload().Company }
	func tag() AccountTag { return initial.Tag() }
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.EnumConstructors) != 1 {
		t.Fatalf("constructors: %d", len(info.EnumConstructors))
	}
	for expression, variant := range info.EnumConstructors {
		if !types.Identical(info.TypeOf(expression), variant.Enum().Named()) || variant.Name() != "Personal" {
			t.Fatalf("literal type: %v; variant: %s", info.TypeOf(expression), variant.Name())
		}
	}
}

// TestEnumSwitch checks variant coverage, payload bindings, and ordinary type rules.
func TestEnumSwitch(t *testing.T) {
	declaration := `type Account enum {
		Personal struct { Name string }
		Business struct { Company string }
	}
	`
	cases := []struct {
		name, source, diagnostic string
	}{
		{"complete", `func label(a Account) string { switch v := a.(type) {
		case Account.Personal: return v.Name
		case Account.Business: return v.Company
		exhaustive:
		} }`, ""},
		{"partial_default", `func label(a Account) string { return switch v := a.(type) {
		case Account.Personal: v.Name
		default: v.Tag(); "other"
		} }`, ""},
		{"grouped", `func label(a Account) Account { switch v := a.(type) {
		case Account.Personal, Account.Business: return v
		exhaustive:
		} }`, ""},
		{"generic", `func label[T interface { Account }](a T) string { return switch v := a.(type) {
		case Account.Personal: v.Name
		case Account.Business: v.Company
		exhaustive:
		} }`, ""},
		{"missing", `func label(a Account) { switch a.(type) { case Account.Personal: exhaustive: } }`, "Business"},
		{"duplicate", `func label(a Account) { switch a.(type) { case Account.Personal: case Account.Personal: } }`, "duplicate"},
		{"wrong_enum", `type Other enum { Personal }
		func label(a Account) { switch a.(type) { case Other.Personal: } }`, "Account"},
		{"grouped_field", `func label(a Account) { switch v := a.(type) { case Account.Personal, Account.Business: _ = v.Name } }`, "Name"},
		{"default_field", `func label(a Account) { switch v := a.(type) { default: _ = v.Name } }`, "Name"},
		{"standalone_variant", `var p Account.Personal`, "Personal"},
		{"marker_body", `func label(a Account) { switch a.(type) { exhaustive: println(1) } }`, "exhaustive"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := checkTGo(t, declaration+test.source)
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

// TestEnumPrivateStorage permits normal values but prevents source access to representation fields.
func TestEnumPrivateStorage(t *testing.T) {
	declaration := `type Account enum { Personal struct { Name string }; Business }
	`
	cases := []struct {
		name, source string
		valid        bool
	}{
		{"zero", `var a = Account{}`, true},
		{"derived", `type Other Account; var a = Account(Other(Account.Personal{Name:"x"}))`, true},
		{"interface", `var a = any(Account{})`, true},
		{"nil_pointer", `var a = (*Account)(nil)`, true},
		{"field", `var a = Account{}.tgoTag`, false},
		{"key", `var a = Account{tgoTag: 1}`, false},
		{"derived_field", `type Other Account; var a = Other{}.tgoTag`, false},
		{"embedded", `type Outer struct { Account }; var a = Outer{}.tgoTag`, false},
		{"conversion", `type Raw struct { tgoTag AccountTag; tgoVariant0 AccountPersonal }; var a = Account(Raw{})`, false},
		{"pointer_conversion", `type Raw struct { tgoTag AccountTag; tgoVariant0 AccountPersonal }; var a = (*Account)(new(Raw))`, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := checkTGo(t, declaration+test.source)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, error=%v", test.valid, err)
			}
		})
	}
}
