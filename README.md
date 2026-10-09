# TGo

TGo adds checked syntax to Go and emits ordinary `*_tgo.go` files.

## Return success

```go
func Name(user User) (string, error) {
	return user.Name,
}
```

The trailing comma adds the final `nil` result.

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

The compiler closes the variant set. Enums support external, internal, adjacent, and untagged
JSON forms.

## Validate construction

```go
type Port int where value > 0 && value < 65536

func PortNumber(text string) (int, error) {
	number := strconv.Atoi(text)!
	port := NewPort(number)!
	return port.Value(),
}
```

TGo generates `NewPort(int) (Port, error)` and `Port.Value() int`.

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

## Build and check

```sh
make build
./bin/tgo build ./...
go test ./...
./bin/tgolint ./...
```

Read the [documentation map](docs/README.md), the
[language specification](docs/spec/README.md), and the
[`tgolint` specification](docs/spec/TGOLINT.md).
