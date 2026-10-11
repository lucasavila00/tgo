# Transpiler use and API

The transpiler reads `.tgo` files in a Go module and emits a Go module.
It uses Go syntax with postfix error propagation.

## Propagate errors

`call()!!` returns the original non-nil error from the current function.
`call()!` wraps it once with the source call name and `%w`. `errors.Is` and
`errors.As` can still find the original error. Both operators remove the
call's final error result from its success values.

```text
package example

import "strconv"

func Parse(text string) (int, error) {
    value := strconv.Atoi(text)!!
    return value, nil
}
```

The containing function must end its result list with `error`. On failure,
other results receive typed zero values. Earlier effects remain; later work
stops. Deferred calls run. A function literal has its own return target.
Propagation works inside calls, assignments, literals, conditional expressions,
loops, switches, ranges, selects, and arguments to `defer` and `go`.

## Command

Run from the repository root:

```sh
go run ./cmd/tgo -o /tmp/generated /path/to/source
go -C /tmp/generated build ./...
go -C /tmp/generated test ./...
```

The source directory must contain `go.mod`. Output includes module files,
ordinary Go source, generated Go, and assets. Go build constraints select which
TGo files to compile, including test packages. Excluded files retain their
`.tgo` source. Relative module replacements keep their dependency paths.
Invalid source and compiler failures cause a panic.

## Go API

Import `github.com/lucasavila00/tgo/pkg/transpiler`.

| Operation | Result |
| --- | --- |
| `Compile(directory string) map[string][]byte` | Emitted module files keyed by relative path; panics on failure |
| `Load(directory, pattern string) (*Source, error)` | One Go package with syntax and type information |
| `source.InsertAfter(site Site, after After) (map[string][]byte, error)` | Formatted Go files keyed by absolute path |

`Load` reads ordinary Go syntax. `Source.Package` contains the loaded
`packages.Package`; `Source.Sites` lists the original expressions. Each `Site`
records its file, byte offsets, and AST kind. `Reason` explains syntax that has
no evaluation in the current function. `ErrorReturn` identifies a function
with one `error` result; `ReturnReason` explains other return signatures.

`After` is a callback with type `func([]ast.Expr) []ast.Stmt`. Its expression
references are the evaluated results, including multiple results from a call.
An assignment destination supplies no value result. The callback must return
valid Go statements for the insertion scope.

For a site inside a function with an `err` binding and one `error` result:

```go
files, err := source.InsertAfter(site, func(results []ast.Expr) []ast.Stmt {
    return []ast.Stmt{
        &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}},
    }
})
```

The insertion executes after the selected expression and before its parent
uses the result. It executes only when the expression executes. The source AST
stays unchanged, so callers can generate independent variants from one load.
An unknown site or a site without a current-function evaluation returns an error.

See the [architecture](arch/transpiler.md) for lowering and package dependencies,
and the [test guide](../tests/README.md) for behavior checks and snapshots.
