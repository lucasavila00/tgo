# TGo

TGo adds checked syntax to Go and emits ordinary `*_tgo.go` files. It preserves
Go packages, imports, types, calls, and the generated ABI.

## Return success or failure

```go
func Name(user User) (string, error) {
	return user.Name,
}
```

The trailing comma adds the final `nil` result.

Use leading commas to return zero values with one error:

```go
func Parse(text string) (int, error) {
	if text == "" { return , ErrEmpty }
	return strconv.Atoi(text)!,
}
```

## Propagate errors with context

```go
func LoadName(repo Repo, id ID) (string, error) {
	user := repo.Find(id)!
	return user.Name,
}
```

`!` returns the error with `repo.Find: ` context. It keeps `errors.Is` and `errors.As` working.

Use `!!` to return the original error without context or wrapping. On success, the expression
yields the call's non-error result:

```go
func LoadName(repo Repo, id ID) (string, error) {
	user := repo.Find(id)!!
	return user.Name,
}
```

## Require non-nil pointers

```go
type Account struct {
	Owner   %User
	Manager *User
}

func OwnerName(account Account) string {
	return account.Owner.Name
}
```

`%User` emits `*User`. `tgolint` proves that each value at this boundary is non-nil.

## Model closed choices

```go
type Account enum {
	Personal struct { Name string }
	Business struct { Company string }
}

account := Account.Personal{Name: "Lucas"}
```

The compiler closes the variant set. A checked tag switch can use `exhaustive:`
to require every variant. Enums support external, internal, adjacent, and
untagged JSON forms. Payload-free enums also have a stable gob form.

## Validate construction

```go
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
	port := Port{number: number}!
	return port,
}
```

The compiler calls `check` for each `Port` literal. The same literal form works from another TGo
package even though all fields are private. A checked value permits field reads in its package,
but TGo rejects field writes and address-taking after construction.

The compiler generates `NewPort(number int) (Port, error)` for Go callers. TGo source must use the
literal form and cannot call this generated ABI. The compiler does not generate field accessors.

## Declare field defaults

```go
type Request struct {
	ID   string
	Tags map[string]string = map[string]string{}
}

func NewRequest(id string) Request {
	return Request{ID: id, ..default}
}
```

Each construction gets a new map.

## Build collections

```go
names := []string{for _, account := range accounts {
	if account.Active { account.Name }
}}

byID := map[ID]Account{for _, account := range accounts {
	account.ID: account
}}
```

Comprehensions emit direct Go loops. They add no iterator or runtime helper.

## Keep values valid

TGo requires explicit variable initialization, complete struct and collection
literals, initialized named results, and proofs for invalid zero values.
Presence checks protect map reads, channel receives, and type assertions.
Bounds checks can prove a reslice safe. `tgolint` checks these rules in TGo
source and in Go code that uses generated TGo models.

Read the [language specification](docs/spec/README.md) and the
[`tgolint` specification](docs/spec/TGOLINT.md) for the complete rules.

## Use TGo packages from Go

Write TGo package tests in `_test.tgo` files. Go build constraints and target
suffixes select active TGo files. Generated production and test Go files stay
beside their TGo source and are committed. Go packages can import a TGo package
and use its original Go types. Generated APIs provide the Go construction
boundary for checked structs and enums.

TGo trusts values that arrive from Go. It adds no runtime wrapper. Run
`tgolint` across both sides of the boundary. See the [user guide](docs/guide/README.md)
and [Go caller guide](docs/guide/GO-CALLERS.md).

## Tools

- `tgo build` checks TGo and writes ordinary Go output.
- `tgofmt` formats TGo and matches `gofmt` for ordinary Go syntax.
- `tgolint` checks TGo source policy and Go use of generated models.
- `pkg/syntax` provides the public, closed TGo syntax tree for source tools.
- `tgonav` provides hover, definition, references, and symbols to the
  [VS Code extension](docs/guide/VSCODE.md).

The extension also provides TGo syntax highlighting and hides generated files
by default.

## Build and check

Each package is either TGo or Go. Do not mix handwritten `.tgo` and `.go`
files in one package. A Go package can import the generated API of a TGo package.

```sh
make build
./bin/tgo build ./...
go test ./...
./bin/tgolint ./...
```

Read the [documentation map](docs/README.md), the
[language specification](docs/spec/README.md), and the
[`tgolint` specification](docs/spec/TGOLINT.md).
