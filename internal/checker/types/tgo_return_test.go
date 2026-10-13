package types_test

import (
	"strings"
	"testing"
)

// TestShortReturnTypes checks valid omitted results with generics, contextual types, and nested expressions.
func TestShortReturnTypes(t *testing.T) {
	for _, source := range []string{
		`func f(e error) (int, string, error) {return .. e}`,
		`func f() (int, string, error) {return 1, "ok" ..}`,
		`func f(e error) error {return .. e}`,
		`func f[T any](e error) (T, error) {return .. e}`,
		`type E interface {error}; func f(e E) (int, E) {return .. e}`,
		`type E interface {error}; func f() (int, E) {return 1 ..}`,
		`type E struct{}; func(E) Error() string {return ""}; func f(e E) (int, E) {return .. e}`,
		`type E struct{}; func(*E) Error() string {return ""}; func f() (int, *E) {return 1 ..}`,
		`func f[E error](e E) (int, E) {return .. e}`,
		`type E struct{}; func(*E) Error() string {return ""}; type C interface ` +
			`{error; ~*E}; func f[T C]() (int, T) {return 1 ..}`,

		`func pair() (int,string) {return 1,"x"}; func f() (int,string,error) {return pair() ..}`,
		`func f(n uint) (uint64, error) {return 1<<n ..}`,
		`func f() (*int, error) {return nil ..}`,
		`func f() (any, error) {return nil ..}`,
		`func f() (int, error) {nil := 3; _=nil; return 1 ..}`,
		`type E interface {Error() string}; type error int; func f(e E) (int,E) {return .. e}`,
		`func f(ok bool) (uint64,error) {return if ok {1} else {2} ..}`,
		`func f(ok bool,e error) (uint64,error) {return if ok {return .. e} else {2} ..}`,
		`func f(ok bool) (int,error) {return .. if ok {nil} else {nil}}`,
		`func read() (int,error) {return 1,nil}; func f() (int,error) {return read()! ..}`,
		`func f() (n int,e error) {inner := func() (int,error) {return 1 ..}; return inner()}`,
		`func f(e error) (int,error) {for {x:=if true {return .. e} else {1}; return x ..}}`,
	} {
		t.Run(source, func(t *testing.T) {
			if _, err := checkTGo(t, source); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestShortReturnTypeErrors checks result counts, types, named-result restrictions, and nil error requirements.
func TestShortReturnTypeErrors(t *testing.T) {
	cases := []struct{ source, diagnostic string }{
		{`func f() (n int,e error) {return 1 ..}`, "requires unnamed results"},
		{`func f(e error) (_ int,_ error) {return .. e}`, "requires unnamed results"},
		{`func f() (a,b int,e error) {return 1,2 ..}`, "requires unnamed results"},
		{`func f() {return 1 ..}`, "requires a final result that implements error"},
		{`func f() (int,int) {return 1 ..}`, "requires a final result that implements error"},
		{`func f() (int,error) {return 1,2 ..}`, "too many return values"},
		{`func f() (int,int,error) {return 1 ..}`, "not enough return values"},
		{`func f() (int,error) {return "x" ..}`, "cannot use"},
		{`func f() (int,error) {return .. 1}`, "cannot use"},
		{`func f(e error) (int,error) {return .. e,e}`, "too many return values"},
		{`type E struct{}; func(E) Error() string {return ""}; func f() (int,E) ` +
			`{return 1 ..}`,
			"requires nil to fit"},

		{`func f[T error]() (int,T) {return 1 ..}`, "requires nil to fit"},
		{`func f() (int8,error) {return 128 ..}`, "overflows"},
		{`func f(xs []int) (int,error) {return xs... ..}`, "cannot end with ..."},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			_, err := checkTGo(t, test.source)
			if err == nil || !strings.Contains(err.Error(), test.diagnostic) || !strings.Contains(err.Error(),
				"main.tgo:") {
				t.Fatalf("got %v, want %s", err, test.diagnostic)
			}
		})
	}
}

// TestShortReturnDiagnosticPositions checks source locations for named-result and overflow errors.
func TestShortReturnDiagnosticPositions(t *testing.T) {
	cases := []struct{ source, position string }{
		{"func f() (value int, err error) {\n\treturn 1 ..\n}", "main.tgo:3:2:"},
		{"func f() (int8, error) {\n\treturn 128 ..\n}", "main.tgo:3:9:"},
	}
	for _, test := range cases {
		_, err := checkTGo(t, test.source)
		if err == nil || !strings.Contains(err.Error(), test.position) {
			t.Fatalf("got %v, want position %s", err, test.position)
		}
	}
}
