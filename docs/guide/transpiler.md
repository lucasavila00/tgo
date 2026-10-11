# Use the TGo compiler

Write TGo in `.tgo` files inside a Go module. Compilation writes ordinary Go
beside those inputs, then checks it with Go. Use Go's normal commands to build,
test, or run the result.

## Return a call's error

`call()!` returns the call's error on failure. On success, the expression
provides the call's results before the error:

```text
package example

import "strconv"

func Parse(text string) (int, error) {
    value := strconv.Atoi(text)!
    return value, nil
}
```

This has the behavior of:

```go
func Parse(text string) (int, error) {
    value, err := strconv.Atoi(text)
    if err != nil {
        return 0, err
    }
    return value, nil
}
```

`call()!!` does the same check but wraps failure with the source call name and
`%w`. For example, a failure from `strconv.Atoi(text)!!` gets `strconv.Atoi`
context. `errors.Is` and `errors.As` can still find the original error.

The enclosing function must end its result list with an error interface. The
original error must fit that result for `!`; the wrapped `error` must fit it for
`!!`. Failure returns zero values for the other results. Source deferred calls
still run. A real function literal has its own return target.

Propagation can appear inside larger expressions. In
`consume(first(), fetch()!, last())`, `first` runs before `fetch`. If `fetch`
fails, `last` and `consume` do not run. Propagation is not allowed inside an
operand of `unsafe.Sizeof`, `unsafe.Alignof`, or `unsafe.Offsetof`, including
inside nested function bodies in that operand.

## Use an if expression as a value

An if value executes one branch and uses its final expression:

```text
name := if active {
    log("active")
    account.Name
} else {
    "inactive"
}
```

An if value requires `else`. Branches can contain statements. A return, defer,
propagation, or jump belongs to the enclosing function. A path that returns or
cannot reach the branch end needs no final value.

An assignment, return, argument, or conversion can supply the result type.
Otherwise, typed branch results must have the same type, with Go's rules for
untyped constants. `nil` cannot establish a type on its own. Use a type or
conversion when the result type is unknown or branch types differ.

Each completing branch supplies one value. An if value is not a constant, so it
cannot be a `const` initializer or an array length. Ordinary if statements keep
Go behavior. A `continue` cannot target a range while its collection is still
being computed.

## Build a slice or map

State the collection type, then the output, range clauses, and filters:

```text
names := []string{user.Name for _, user := range users if user.Active}
byID := map[ID]User{user.ID: user for _, user := range users}
pairs := [][2]int{[2]int{x, y} for _, x := range xs for y := range x}
```

Each range uses Go's binding and iteration rules. A false filter skips the
remaining clauses for that visit. Later clauses and the output can use earlier
bindings. Bindings stay inside the comprehension.

The result is a fresh, non-nil slice or map, including an empty result. Maps
evaluate the key before the value. Later equal keys replace earlier entries.
Outputs follow Go assignment rules for the declared element, key, and value
types. Named collection types and type parameters with a common slice or map
shape are supported.

Outputs, sources, and filters can use `if` values and propagation. Returns and
defers belong to the enclosing function. In a filter or output, an unlabeled
`break` or `continue` targets the current generator. In a generator's source,
a jump can target only an already active outer construct.

## Compile and run

From the repository root:

```sh
make prepare
go run ./cmd/tgo /path/to/source
go -C /path/to/source build ./...
go -C /path/to/source test ./...
```

The argument can be a module subdirectory. The compiler finds its `go.mod` and
works from the module root. There is no output-directory option.

| TGo source | Generated Go |
| --- | --- |
| `name.tgo` | `name_tgo.go` |
| `name_test.tgo` | `name_tgo_test.go` |
| `name_linux_amd64_test.tgo` | `name_tgo_linux_amd64_test.go` |

Go build constraints and platform rules select the files, including test
variants. Existing generated snapshots are replaced from their TGo inputs.
Generated files for excluded inputs are removed; the `.tgo` inputs remain.
Nested modules, vendor dependencies, and data directories are not compiler
outputs. Assets, headers, and local dependencies stay at their original paths.

The compiler writes the source-checking and final versions at the same generated
paths. It creates no temporary module. A failure can leave a source-checking
version or completed output at those paths. Correct the TGo input and compile
again. Invalid source and compiler failures panic. Normal Go module loading can
update `go.mod` and `go.sum`.

See [the driver](../../internal/driver/readme.md) for each step.

## Go API

See [pkg/transpiler](../../pkg/transpiler/readme.md) for `Compile`, `Load`,
and `InsertAfter`, including expression-insertion examples.
