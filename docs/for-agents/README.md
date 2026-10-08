# Use tgo

Copy [these rules](AGENTS.md) into a tgo user repository's `AGENTS.md`.

Write small business packages in `.tgo` files. Keep tests and other Go code in `.go` files.
Use one Go module. Keep normal Go imports, package names, and tests.

## Build and check

```sh
tgo build ./...
go test ./...
```

`tgo build` checks the current package. `tgo build ./...` checks packages below it.
Local tgo imports build first. Output goes beside each input: `model.tgo` becomes `model_tgo.go`.
Commit each generated file beside its tgo source. Do not edit generated files.
A failed build restores the output files that it changed.
The compiler removes generated files after their source is deleted.
Ignore `.tgo.lock`. It serializes builds in one module.

Use Go build constraints and target suffixes on tgo files.
For example, `store_linux.tgo` emits `store_tgo_linux.go`.
The compiler keeps outputs for other targets when it builds the current target.
It ignores `_test.tgo`, hidden, `_`, `testdata`, and `vendor` paths.
It stops at nested Go modules.

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

## Call Go

Import Go packages and call them directly. Keep their types, callbacks, and error results.
Go code can call generated functions. Variant constructors use these names:

```go
account := model.NewAccountPersonal(model.AccountPersonal{Name: "Lucas"})
quantity, err := model.NewQuantity(3)
```

Check `err` before using `quantity`. Go can create invalid zeros and change shared data.
The generated code trusts Go callers. There are no boundary scans or read guards.

[Language specification](../spec/README.md).
