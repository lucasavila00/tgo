# TGo

## Repository status

This repository is mothballed while we develop compilerv2 in
[#248](https://github.com/lucasavila00/tgo/issues/248) and
[#247](https://github.com/lucasavila00/tgo/issues/247). Only the README syntax
examples and project documentation remain. The code, tests, build tools, CI, and architecture
documents have been removed. Git history retains them.

TGo is Go with extra compile-time checks and shorter error handling. Its
`.tgo` syntax uses normal Go packages, imports, types, and calls.

The examples below describe the language syntax. This checkout has no compiler
or installation procedure.

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

## Build slices and maps with comprehensions

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

Use one expression for a slice result and `key: value` for a map result. A later
entry with the same map key replaces the earlier value.

## Use TGo with Go

Each package is either TGo or Go. Do not mix handwritten `.tgo` and `.go`
files in one package.

TGo emits normal Go types. A Go package can import a TGo package and use its
generated API. TGo adds no automatic runtime guard at the Go boundary. Go code
can bypass generated constructors and other TGo checks.

## Project direction

See the [project problem](docs/problem/README.md) for the language design goals.
