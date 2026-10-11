# Language vision

This document describes the intended TGo language and its syntax. TGo uses
Go packages, imports, types, and calls. See [transpiler use and API](transpiler.md)
for the current compiler.

## Return success or failure

```text
package greeting

import "errors"

var ErrEmptyName = errors.New("empty name")

func Greeting(name string) (string, error) {
	if name == "" {
		return , ErrEmptyName
	}
	return "Hello, " + name,
}
```

## Return and propagate errors

A trailing comma supplies the final `nil` result. A leading comma supplies the
zero value for a result before an error.

```text
func Name(user User) (string, error) {
	return user.Name,
}

func Parse(text string) (int, error) {
	if text == "" {
		return , ErrEmpty
	}
	return strconv.Atoi(text)!,
}
```

If `strconv.Atoi` returns an error, `!` immediately returns from `Parse` and
adds the call name to the error. `errors.Is` and `errors.As` continue to work.

## Require a non-nil pointer

```text
type Account struct {
	Owner   %User
	Manager *User
}

func OwnerName(account Account) string {
	return account.Owner.Name
}
```

`%User` means a pointer that must not be nil. Generated Go uses `*User`.
A value supplied to `%User` must be proven non-nil.

## List every allowed form

```text
type Account enum {
	Personal struct { Name string }
	Business struct { Company string }
}

account := Account.Personal{Name: "Lucas"}
```

An enum lists all allowed forms of a value. An `exhaustive:` switch must handle
every form.

## Validate values when they are created

```text
type Port struct {
	number int
} checked

func (value Port) check() (Port, error) {
	if value.number < 1 || value.number > 65535 {
		return Port{}, ErrInvalidPort
	}
	return value,
}

func ParsePort(text string) (Port, error) {
	number := strconv.Atoi(text)!
	return Port{number: number}!,
}
```

A checked `Port` literal runs `check` and returns `(Port, error)`. After
construction, TGo rejects field writes, field addresses, and pointer-method
access that can change a field. A checked field must be boolean, numeric,
string, or another checked struct by value. Named types and aliases follow the
same rule. Pointers, collections, arrays, functions, channels, interfaces,
enums, and ordinary structs are not valid checked fields.

## List and map comprehensions

A slice comprehension produces a list of values. An `if` filters the input:

```text
names := []string{for _, account := range accounts {
    if account.Active { account.Name }
}}
```

A map comprehension produces a key and value for each input:

```text
byID := map[ID]Account{for _, account := range accounts {
    account.ID: account
}}
```

Use one expression for a list result and `key: value` for a map result. Filters
also work in map comprehensions. Nested ranges produce combinations:

```text
pairs := [][2]int{for _, account := range accounts {
    for _, sale := range account.Sales {
        [2]int{len(account.Name), sale}
    }
}}

activeByID := map[ID]Account{for _, account := range accounts {
    if account.Active { account.ID: account }
}}
```

Each range uses one or two names with `:=`. The deepest block supplies one
result. An equal map key replaces its earlier value; map iteration order is
unspecified. Each range source runs once when its containing loop reaches it.
Accepted results run once, with map keys before values. An empty list is non-nil.
Postfix error propagation in a source, filter, or result returns from the
surrounding function.

## Use TGo with Go

Each package is either TGo or Go. Do not mix handwritten `.tgo` and `.go`
files in one package.

TGo emits normal Go types. A Go package can import a TGo package and use its
generated API. TGo adds no automatic runtime guard at the Go boundary. Go code
can bypass generated constructors and other TGo checks.

## Project direction

See the [project problem](problem/README.md) for the language design goals.
