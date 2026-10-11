# How compilation works

TGo turns `.tgo` source into ordinary Go. The driver runs the steps; the
frontend reads added syntax; the checker validates the source; the lowerer
builds the Go statements that implement it.

Compilation runs in the original Go module. It uses each final `_tgo.go` path
twice: first for a source-checking version, then for the final implementation.
It creates no temporary module or separate temporary source file.

## Start with one file

Suppose `service.tgo` contains this function. `fetch` returns `(Value, error)`:

```text
func run() error {
    consume(fetch()!)
    return nil
}
```

After compilation, `service.tgo` still contains that source. A new
`service_tgo.go` contains ordinary Go. Its essential behavior is:

```go
func run() error {
    value, err := fetch()
    if err != nil {
        return err
    }
    consume(value)
    return nil
}
```

The actual output uses fresh names and explicit checked types. The compiler
must call `fetch` once, return its error on failure, and call `consume` only on
success.

## The complete process

[`driver.Compile`](../../../internal/driver/compile.go) calls these steps:

```text
findModule → readModule → prepareModule → selectSources
    → checkSources → lowerModule → checkGenerated
```

### 1. Find the module

`findModule` runs `go env GOMOD` from the supplied directory. The selected
`go.mod` gives the module root. You can supply a subdirectory, but the driver
compiles from that root. A directory outside a Go module is an error.

### 2. Read the source

`readModule` lists module paths, then reads owned `.go` and `.tgo` source. It
does not read unrelated asset contents. It skips `.git` and `.worktrees` while
walking the tree.

A `.tgo` input controls its generated output. If `service_tgo.go` already
exists, the driver excludes it from the source set. Its old contents must not
affect the new compilation.

The driver does not own inputs in nested modules, `vendor`, `testdata`, or
directories whose names start with `.` or `_`. Their TGo files and snapshots
stay on disk. A nested `go.mod` marks a separate module.

### 3. Write the source-checking version

`prepareModule` writes the original TGo bytes to each owned generated Go path.
At this point, `service_tgo.go` contains TGo syntax, including `!` and `if`
values. It is not yet the final Go implementation.

Go selects files with `.go` names. These paths let its package tools apply
normal file-selection rules. Filename conversion keeps test and platform
suffixes at the end:

| Input | Generated path |
| --- | --- |
| `service.tgo` | `service_tgo.go` |
| `service_test.tgo` | `service_tgo_test.go` |
| `service_linux.tgo` | `service_tgo_linux.go` |
| `service_linux_amd64_test.tgo` | `service_tgo_linux_amd64_test.go` |

Preparation also reserves source names and records each generated path's
original TGo path. Later steps use this record to identify TGo source and report
its original diagnostic locations.

### 4. Let Go select packages and files

`selectSources` calls `go/packages` for `./...`, with tests enabled. It requests
build metadata, not source type checking.

Go supplies file lists, build constraints, platform selection, imports, test
package variants, and locations of compiled dependency information. The driver
does not implement those build rules itself.

Ownership and selection are different. A Windows-specific TGo input belongs to
the module, but Go excludes it from a Linux build. A vendored TGo input belongs
to a dependency, so the driver does not stage it at all.

### 5. Check the source

`checkSources` creates a private checker loader. It reads selected staged files
through the frontend and ordinary Go files through Go's parser.

The frontend represents TGo expressions with marker calls in the syntax tree.
A marker identifies a compiler operation; it is not a runtime helper. The
private checker understands what the marker means.

If `fetch` returns `(Value, error)`, the checker gives `fetch()!` the type
`Value`. It can then check `consume(fetch()!)` directly. For:

```text
outer(inner()!)!
```

it checks the inner call, obtains its success type, and uses that type to check
the outer call. No package reload is needed for each operator.

The checker also checks `if` branch values, scopes, jumps, and the surrounding
type. Returns and defers inside a branch belong to the source function.

The loader checks source dependencies as needed and caches each package ID.
Test variants have separate IDs. External dependencies use compiled export
data. All source packages pass this check before final implementations are
written. See [source checking](source-checking.md) for these rules.

### 6. Build and write ordinary Go

`lowerModule` gives the checked syntax, types, bindings, and operation records
to the lowerer. Lowering means replacing TGo expressions with Go statements
that have the same behavior.

The lowerer decides where calls, saved values, branches, and error returns
must execute. The driver writes the resulting bytes only to paths owned by TGo
inputs. Ordinary `.go` source is not overwritten.

A staged input with no TGo operations already contains ordinary Go and needs
no rewrite. After lowering, the driver removes owned generated paths that Go
did not select, such as Windows-specific outputs in a Linux build.

See [how lowering preserves execution](correctness.md) for calls, assignments,
loops, types, and returns.

### 7. Check the final Go

`checkGenerated` loads the final module with stock Go type checking. This checks
generated files, ordinary Go, imports, and test variants together.

The source check proves that the input satisfies TGo's type rules. The final
Go check catches invalid emitted Go. Behavior tests are also needed: valid Go
can still have the wrong evaluation order.

`Compile` returns no file contents. The generated files are already on disk.
Invalid source and compiler failures panic.

## Files on disk

| File | During source checking | After success |
| --- | --- | --- |
| `service.tgo` | Original TGo | Original TGo |
| `service_tgo.go` | Original TGo at a Go path | Final ordinary Go |
| Excluded owned generated file | Staged source | Removed |
| Ordinary Go and assets | At original paths | At original paths |

Embedded assets, cgo headers, vendor files, and local dependencies remain
available at their original paths. Go commands can update `go.mod` and
`go.sum` through normal module loading. Relative replacements stay relative.

Compilation is not an atomic disk update. A failed check can leave TGo text or
some completed output at generated paths. The next compilation replaces owned
outputs from their `.tgo` inputs.

## Where to read the code

| Package | Job |
| --- | --- |
| [`cmd/tgo`](../../../cmd/tgo/readme.md) | Read the source-directory argument and call `Compile`. |
| [`pkg/transpiler`](../../../pkg/transpiler/readme.md) | Expose compilation and the separate expression-insertion API. |
| [`internal/driver`](../../../internal/driver/readme.md) | Run the module steps and write generated files. |
| [`internal/frontend`](../../../internal/frontend/readme.md) | Prepare added syntax for Go parsing and record source positions. |
| [`internal/checker`](../../../internal/checker/readme.md) | Check source types and import dependency types. |
| [`internal/controlflow`](../../../internal/controlflow/readme.md) | Determine whether an if branch can reach its value or end. |
| [`internal/lowering`](../../../internal/lowering/readme.md) | Emit Go with the same evaluation, scope, and control flow. |

Within the driver, start with `compile.go`. `module.go` handles source ownership
and disk stages. `propagation.go` coordinates selection, checking, and lowering
for both propagation and if values. `filename.go` preserves filename suffixes.
`unsafe_propagation.go` rejects propagation inside unsafe size and offset
operands. Package comments and the README contain no runtime logic.

## Terms used in these docs

| Term | Meaning |
| --- | --- |
| AST or syntax tree | Go nodes that represent expressions, statements, and declarations. |
| Marker | A node that records a TGo operation for the checker and lowerer. |
| Private checker | The compiler's patched copy of Go's type checker. |
| Stock Go checker | The unmodified Go type checker used for final output. |
| Export data | Compiled information about a dependency's types and declarations. |
| Package ID | Go's identity for one package, including a particular test variant. |
| Binding | The specific declaration to which a name refers. |
| Contextual type | A type supplied by an assignment, return, argument, or conversion. |
| Temporary variable | A generated local that saves a value; not a temporary file. |

For commands and syntax, read the [user guide](../../guide/transpiler.md).
