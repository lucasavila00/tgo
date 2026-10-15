# Language vision

This document describes the intended TGo language and its syntax. TGo uses
Go packages, imports, types, and calls. See [compiler use](../guide/transpiler.md)
for the current compiler.

## Return success or failure

```text
package greeting

import "errors"

var ErrEmptyName = errors.New("empty name")

func Greeting(name string) (string, error) {
	if name == "" {
		return ..., ErrEmptyName
	}
	return "Hello, " + name, ...
}
```

## Return and propagate errors

A last `...` supplies the final `nil` result. A first `...` supplies zero
values for the success results. A comma separates it from supplied values.

```text
func Name(user User) (string, error) {
	return user.Name, ...
}

func Parse(text string) (int, error) {
	if text == "" {
		return ..., ErrEmpty
	}
	return strconv.Atoi(text)!, ...
}
```

If `strconv.Atoi` fails, `!` returns the same error from `Parse`. Use `!!` to add
the call name. `errors.Is` and `errors.As` can still find the original error.

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
	return value, ...
}

func ParsePort(text string) (Port, error) {
	number := strconv.Atoi(text)!
	return Port{number: number}!, ...
}
```

A checked `Port` literal runs `check` and returns `(Port, error)`. After
construction, TGo rejects field writes, field addresses, and pointer-method
access that can change a field. A checked field must be boolean, numeric,
string, or another checked struct by value. Named types and aliases follow the
same rule. Pointers, collections, arrays, functions, channels, interfaces,
enums, and ordinary structs are not valid checked fields.

## List and map comprehensions

Build a slice or map with generators, filters, and output expressions. Combine
those steps in direct loops that write into one result collection.

This feature is not implemented. The [comprehension proposal](../adr/collection-comprehensions.md)
owns the syntax alternatives, behavior, optimization rules, and validation plan.

## Use TGo with Go

Each package is either TGo or Go. Do not mix handwritten `.tgo` and `.go`
files in one package.

TGo emits normal Go types. A Go package can import a TGo package and use its
generated API. TGo adds no automatic runtime guard at the Go boundary. Go code
can bypass generated constructors and other TGo checks.

## Project direction

See the [project problem](README.md) for the language design goals.
