# Use TGo

Copy [these rules](../for-agents/AGENTS.md) into a TGo user repository's `AGENTS.md`.

Write small business packages in `.tgo` files. Keep tests and other Go code in `.go` files.
Use one Go module. Keep normal Go imports, package names, and tests.

## Build and check

```sh
tgo build ./...
go test ./...
tgolint ./...
```

`tgo build` checks the current package. `tgo build ./...` checks packages below it.
Local TGo imports build first. Output goes beside each input: `model.tgo` becomes `model_tgo.go`.
Commit each generated file beside its TGo source. Do not edit generated files.
The generated output name identifies its TGo source.
`tgolint` recompiles the package in memory and verifies the complete output before it trusts model
facts.
Run `tgo build` again when an integrity diagnostic reports stale or changed output.
A failed build restores the output files that it changed.
The compiler removes generated files after their source is deleted.
Ignore `.tgo.lock`. It serializes builds in one module.

Use Go build constraints and target suffixes on TGo files.
For example, `store_linux.tgo` emits `store_tgo_linux.go`.
The compiler keeps outputs for other targets when it builds the current target.
It ignores `_test.tgo`, hidden, `_`, `testdata`, and `vendor` paths.
It stops at nested Go modules.

## Parse TGo source

Use the public `tgo/pkg/syntax` package when a tool must read TGo source. Call `ParseFile` with a
Go file set. Use `syntax.Inspect` or `syntax.Walk` to visit the closed TGo node enums. Use
`Extensions`, `Parent`, `Children`, and `AttachedComments` for direct queries.

Treat a parsed tree as read-only. Its traversal indexes describe the tree at parse time. Match the
`Expression`, `Statement`, `Declaration`, and `Specification` enums. The public tree does not
contain `go/ast` nodes.

## Declare and use business types

```text
package model

type Port struct { number int } checked

func (value Port) check() (Port, error) {
    if value.number < 1 { return Port{}, ErrInvalidPort }
    return value,
}

type Account enum {
    Personal struct { Name string }
    Business struct { Company string }
}

func Label(account Account) string {
    switch account.Tag() {
    case AccountTagPersonal: return account.PersonalPayload().Name
    case AccountTagBusiness: return account.BusinessPayload().Company
    exhaustive:
    }
}
```

Construct with `Account.Personal{Name: "Lucas"}`. `exhaustive:` requires one case for each declared
variant. Use `default:` when the switch needs fallback behavior. Duplicate tags and missing
exhaustive cases fail compilation. Do not use `fallthrough` or select generated enum methods through an interface.
Type aliases can construct variants. Go name resolution selects the aliased type.
Do not shadow generated payload, constructor, or default helper names at a construction.
The value uses a tag and typed Go fields. Reads do not run validation.

Each `Port` literal returns `(Port, error)`. Use postfix `!` to handle the error. All fields must be
private, and `check` must have the exact value-receiver signature shown above. The same literal
works across TGo packages. Go callers use the generated `NewPort` function. TGo source cannot call
that function. The compiler does not generate a field accessor. TGo permits package-local field
reads, but it rejects field changes and address-taking after construction.

## Supply initial values

```text
type Request struct {
    ID string
    Tags map[string]string = map[string]string{}
}

func NewRequest(id string) Request {
    return Request{ID: id, ..default}
}
```

Supply every required field. Use `..default` to select declared defaults.
Explicit fields run first. Selected defaults run in declaration order.
Each map literal makes a fresh map. Struct copies keep Go reference aliases.
Variables need an initializer. Array and slice literals must have no missing indices.
Assign named results before a read or bare return.

Use `%T` when a pointer must not be nil. Keep `*T` when nil is a valid value.

```text
func Save(account %Account) error
```

Check a possibly nil pointer before you pass it to `%T`. For `map[K]%T` and `chan %T`, use comma-ok
or another proof before use. A pointer type assertion needs both `ok` and a nil check. `%T`
becomes `*T` in generated Go and adds no runtime check.

## Call Go

Import Go packages and call them directly. Keep their types, callbacks, and error results.
Go code can call generated enum functions and generated checked struct constructors:

```go
account := model.AccountPersonal{Name: "Lucas"}.Account()
port, err := model.NewPort(3)
```

Check `err` before you use `port`. TGo trusts values from Go. TGo source uses checked literals,
including for types from another package. It cannot call the generated `NewPort` Go ABI.

Use postfix `!` when a call returns Go values followed by `error` and the current function also
ends in `error`:

```text
account := repo.Find(id)!
key, value := index.Entry(id)!
store.Flush()!
```

On failure, TGo adds the call name, wraps the cause, and returns zero values with the error.
Use `call()!!` to return the original error with no context or wrapper allocation.
Use a normal error check when the caller must recover, change the error, or add runtime data.

[Language specification](../spec/README.md).
[Go caller checks](GO-CALLERS.md).
