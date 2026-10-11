# Expression lowering

This package implements [the expression-lowering plan](../docs/adr/expression-lowering/what.md)
for [#248](https://github.com/lucasavila00/tgo/issues/248).
It reads ordinary Go. The `!` and `!!` frontend in #247 will use this engine.

`Load(directory, pattern)` loads one package with Go syntax and type information.
`Source.Sites` lists every expression by file, byte offsets, and syntax kind.
`Reason` identifies syntax without a current-function evaluation. `ErrorReturn`
identifies functions with one `error` result; `ReturnReason` records other
return signatures. Function literals have their own return target.

`InsertAfter(site, callback)` returns formatted files keyed by absolute path.
It keeps the input AST unchanged. The callback receives references to the
expression's results, including tuple results. An assignment destination has
no value result. The callback must supply valid Go statements for that scope.

For an expression in a function with an `err` binding and one `error` result:

```go
files, err := source.InsertAfter(site, func(results []ast.Expr) []ast.Stmt {
    return []ast.Stmt{
        &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}},
    }
})
```

The insertion runs after evaluation and before the surrounding operation uses
the result. It is skipped when the expression is skipped. For example, insertion
after `right()` in `use(left(), right())` produces this order:

```go
leftValue := left()
rightValue := right()
return err
use(leftValue, rightValue)
```

The final call remains in the output so source bindings and label targets remain
valid. It cannot execute after the return. Earlier effects remain; later work
is skipped. Normal deferred calls still run.

| Code | Purpose |
| --- | --- |
| `source.go` | Load types and inventory source expressions |
| `expressions.go` | Save values and keep conditional evaluation |
| `places.go` | Save assignment operands without reading the destination |
| `statements.go`, `control.go`, `select.go` | Keep scope, branches, loop control, and communication timing |
| `insert.go` | Build and format new files |

# Verification

```sh
go -C compilerv2 test ./... -run '^TestProof' -count=1
```

The manifest records expression sites, including excluded syntax and incompatible
return signatures. `TestProofCompile` compiles neutral and return variants.
`TestProofNeutral` runs marker variants against handwritten effect and value
traces and insertion positions across the corpus. `TestProofReturn` checks
first and second visits, early exits, deferred calls, closure return boundaries,
and the identity of the returned error.

Focused execution tests also cover assignments, short circuits, closures,
loops, switches, ranges, selects, defers, jumps, tuples, Boolean conversions,
array storage, and pointer receivers. Hosted CI keeps generated variants and
their test results as artifacts.
