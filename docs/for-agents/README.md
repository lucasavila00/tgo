# Use tgo

Copy [these rules](AGENTS.md) into a tgo user repository's `AGENTS.md`.

Write small business packages in `.tgo` files. Keep tests and other Go code in `.go` files.
Use one Go module. The compiler is written in Go. It does not compile itself.

## Build and check

```sh
make install-tools
export PATH="$(go env GOPATH)/bin:$PATH"
make ci
make build
cd /path/to/your/go/module
/path/to/go2/bin/tgo build ./...
go test ./...
```

`tgo build` checks the current package. `tgo build ./...` checks packages below it.
Local tgo imports build first. Output goes beside each input: `model.tgo` becomes `model_tgo.go`.
Do not edit generated files. A failed build restores the output files that it changed.

CI checks Go style, 100-character Go lines, formatting, Go tests, end-to-end tests, and Markdown width.
The Python tests build the CLI, compile temporary Go modules, and run their Go tests.

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

## Call Go

Import Go packages and call them directly. Keep their types, callbacks, and error results.
Go code can call generated functions. Variant constructors use these names:

```go
account := model.NewAccountPersonal(model.AccountPersonal{Name: "Lucas"})
quantity, err := model.NewQuantity(3)
```

Check `err` before using `quantity`. Go can create invalid zeros and change shared data.
The generated code trusts Go callers. There are no boundary scans or read guards.

## Checkpoint status

The first checkpoint tests two generated packages, Go generic calls, matches, constructor errors,
fresh defaults, and shared maps. The committed model output must match a fresh compile.

The next checkpoint adds nested matches, presence tests for maps, channels, and assertions,
bounded reslicing, overlapping copy, and map clearing. Sixteen invalid-source cases are tested.
A changed bound loses its prior proof. No new runtime bound guard is generated.

The parser checkpoint separates declarations, fields, and expression rewrites.
Field parsing uses Go syntax data for types and tags. CI checks 100-character Go lines,
complexity, duplication, error handling, formatting, and unused code.

The FFI checkpoint compiles direct calls to Go interfaces, methods, generics, callbacks,
channels, named types, variadic functions, pointers, and errors. Tests keep error identity,
typed nil behavior, aliases, and callback mutation. A tested enum match allocates no heap data.

The implementation is still in progress. More control-flow forms, Go type forms, source error
positions, package build edge cases, and cost comparisons need more tests and work.
Do not treat a passing initial test suite as proof of every rule in the proposal.

[Language rules](../tgo/README.md). [Detailed rules](../research/go/tgo/notes.md).
