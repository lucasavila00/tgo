# Use tgo

Copy [these rules](AGENTS.md) into a tgo user repository's `AGENTS.md`.

Write small business packages in `.tgo` files. Keep tests and other Go code in `.go` files.
Use one Go module. Keep normal Go imports, package names, and tests.

## Build and check

```sh
tgo build ./...
go test ./...
tgolint ./...
```

`tgo build` checks the current package. `tgo build ./...` checks packages below it.
Local tgo imports build first. Output goes beside each input: `model.tgo` becomes `model_tgo.go`.
Commit each generated file beside its tgo source. Do not edit generated files.
A generated model file contains versioned integrity metadata for its source and Go body.
`tgolint` verifies generated declarations before it trusts model facts.
Run `tgo build` again when an integrity diagnostic reports stale or changed output.
A failed build restores the output files that it changed.
The compiler removes generated files after their source is deleted.
Ignore `.tgo.lock`. It serializes builds in one module.

Use Go build constraints and target suffixes on tgo files.
For example, `store_linux.tgo` emits `store_tgo_linux.go`.
The compiler keeps outputs for other targets when it builds the current target.
It ignores `_test.tgo`, hidden, `_`, `testdata`, and `vendor` paths.
It stops at nested Go modules.

## Parse tgo source

Use the public `tgo/pkg/syntax` package when a tool must read TGo source. Call `ParseFile` with a
Go file set. Use `syntax.Inspect` or `syntax.Walk` to visit the closed TGo node enums. Use
`Extensions`, `Parent`, `Children`, and `AttachedComments` for direct queries.

Treat a parsed tree as read-only. Its traversal indexes describe the tree at parse time. Match the
`Expression`, `Statement`, `Declaration`, and `Specification` enums. The public tree does not
contain `go/ast` nodes.

## Declare and use business types

```text
package model

type Quantity int where value > 0

type Account enum {
    Personal struct { Name string }
    Business struct { Company string }
}

func Label(account Account) string {
    match account {
    case Personal(person): return person.Name
    case Business(company): return company.Company
    }
}
```

Construct with `Account.Personal{Name: "Lucas"}`. Cover every variant in `match`.
Use `_` to discard a payload. Duplicate or missing cases fail compilation.
Do not use `fallthrough` in a match or select `Tgo*` methods through an interface.
Type aliases can construct variants. Go name resolution selects the aliased type.
Do not shadow generated payload, constructor, or default helper names at a construction.
The value uses a tag and typed Go fields. Reads do not run validation.

Call `NewQuantity(input)` and check its error. Use `quantity.Value()` for the underlying value.
Direct wrapper literals fail. The compiler does not prove that callers check constructor errors.

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
Go code can call generated functions. Variant constructors use these names:

```go
account := model.NewAccountPersonal(model.AccountPersonal{Name: "Lucas"})
quantity, err := model.NewQuantity(3)
```

Check `err` before using `quantity`. TGo trusts values from Go. Go callers must follow the
constructor and accessor rules.

Use postfix `!` when a call returns Go values followed by `error` and the current function also
ends in `error`:

```text
account := repo.Find(id)!
key, value := index.Entry(id)!
store.Flush()!
```

On failure, TGo adds the call name, wraps the cause, and returns zero values with the error.
Use a normal error check when the caller must recover, change the error, or add runtime data.

[Language specification](../spec/README.md).
[Go caller checks](TGOLINT.md).
