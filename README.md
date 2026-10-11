# TGo

## Repository status

This repository is mothballed while we develop the compiler spikes in
[#248](https://github.com/lucasavila00/tgo/issues/248) and
[#247](https://github.com/lucasavila00/tgo/issues/247). Only documentation and
the specification remain. The code, tests, build tools, CI, and architecture
documents have been removed. Git history retains them.

The guides below describe the removed implementation. Their build and test
commands are not available in this checkout.

TGo is Go with extra compile-time checks and shorter error handling. You write
`.tgo` files. `tgo build` creates ordinary `.go` files that the Go toolchain can
build and test.

TGo uses normal Go packages, imports, types, and calls. It does not require a
runtime library or replace the Go toolchain.

## Build TGo

TGo requires Go 1.27.

```sh
git clone https://github.com/lucasavila00/tgo.git
cd tgo
make build
export PATH="$PWD/bin:$PATH"
```

The command builds `tgo`, `tgofmt`, `tgolint`, and `tgonav` in `bin/`.

## Try it

Create a separate Go module:

```sh
cd ..
mkdir tgo-example
cd tgo-example
go mod init example.com/greeting
```

Save this file as `greeting.tgo`:

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

Build the TGo source, then use the normal Go tools:

```sh
tgo build ./...
go test ./...
tgolint ./...
```

`tgo build` writes `greeting_tgo.go` beside `greeting.tgo`. Commit generated Go
files with their TGo source. Do not edit generated files. Write TGo package
tests in `_test.tgo` files; `tgo build` writes `_tgo_test.go` files for the Go
tool.

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
`tgolint` reports code that might supply nil to `%User`.

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

## Use TGo with Go

Each package is either TGo or Go. Do not mix handwritten `.tgo` and `.go`
files in one package.

TGo emits normal Go types. A Go package can import a TGo package and use its
generated API. TGo adds no automatic runtime guard at the Go boundary. Go code
can bypass generated constructors and other TGo checks. Run `tgolint` on TGo
source and Go callers.

See the [user guide](docs/guide/README.md) and the
[Go caller guide](docs/guide/GO-CALLERS.md) for the complete workflow.

## Tools

- `tgo build` checks TGo source and writes Go output.
- `tgofmt` formats `.tgo` files.
- `tgolint` checks TGo rules in TGo source and Go callers.
- The [VS Code extension](docs/guide/VSCODE.md) provides syntax highlighting,
  hover information, navigation, and symbols.

## Contribute

Use the specification and user guides to define language behavior. Keep
compiler experiments separate from the removed implementation.

## Learn more

- [User guide](docs/guide/README.md)
- [Language reference](docs/spec/README.md)
- [`tgolint` reference](docs/spec/TGOLINT.md)
