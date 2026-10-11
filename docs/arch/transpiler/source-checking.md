# How source checking works

The driver checks TGo before it generates the final Go implementation. This
lets it validate the source expression that the user wrote, including its types,
scope, and jumps.

Go's standard parser does not accept postfix propagation or if values. The
frontend prepares those parts for parsing. A private copy of Go's type checker
then applies their TGo rules. The parser itself is not patched.

## Parse source and record TGo operations

[`frontend.ParseFile`](../../../internal/frontend/parse.go) returns a Go syntax
tree and records for propagation and if expressions. A marker call in the tree
identifies each operation. No helper declaration is compiled or called at runtime.

For propagation, the frontend first replaces `!` or `!!` with a named comment.
Go can parse the surrounding call. The frontend then replaces that call in the
AST with a marker and records the operator, call name, and original position.

For if values, the frontend prepares clear expression positions, such as
`return if` or `use(if ...)`, before parsing the complete file. It uses parsed
syntax to resolve ambiguous positions, such as a colon that can introduce a
label, map value, case body, or slice bound. Independent expressions share a
preparation edit. Nested expressions can need more rounds.

Offsets map prepared text back to original source bytes. The generated physical
path controls build selection, while diagnostics identify the original `.tgo`
file. A placeholder function literal carries if-branch syntax in the AST; the
checker treats those statements as part of the enclosing source function.

## Give propagation its success types

Suppose `inner` returns `(Value, error)` and `outer` accepts `Value`:

```text
outer(inner()!)!
```

The checker checks `inner`, validates its final error result, and gives
`inner()!` the type `Value`. It then checks `outer` with that argument and applies
the outer propagation rule. This is one source check, with no reload for each
operator.

A call with several success results keeps those results. A call that returns
only an error leaves no success value. The enclosing function must end its
results with a suitable error interface. For `!`, the original error type must
fit that return. For `!!`, the wrapped `error` type must fit it.

These checks establish types and valid return behavior. The lowerer later
inserts the actual calls, checks, and return statements.

## Check if values in their source context

Each completing branch supplies one value. An assignment, return, argument, or
conversion can provide the required type:

```text
var name string = if ok { "yes" } else { "no" }
```

The checker checks each branch against `string`. Without a supplied type, it
uses the branch types and Go's constant rules. `nil` alone supplies no type.

Generic calls can get a type from another argument. For a function
`pair[T any](a, b T)`, this call can infer `T` as `*int` from `pointer`:

```text
pair(if ok { nil } else { nil }, pointer)
```

Branches that return or jump out supply no value for inference. Control-flow
analysis determines which branch endings can be reached. Source returns,
defers, labels, and jumps keep their enclosing function and targets.

## Check imports once per package ID

[`checker.Loader`](../../../internal/checker/package.go) checks source
dependencies as needed and caches the result by Go's package ID. A package and
its test variant have different IDs, so they receive separate checks.

External dependencies are read from compiled export data: type and declaration
information that Go has already produced. The matching private importer gives
them the same kind of type objects as the private source checker. The loader
reads the selected Go architecture once and uses its word sizes for checking.

The public `Load` API is separate. It retains stock `packages.Package` and
`go/types` objects. Private checking uses the same ordinary Go syntax but keeps
its type objects separate. Lowering uses private checked information.

After all source checks pass, lowering writes ordinary Go. One final stock Go
load checks that output. There are two package-load phases: build metadata and
final Go checking. The number does not grow with the number of operators.

## Maintain the private checker

`make prepare` generates private Go source from a pinned version and a tracked
patch. The checker and its export-data reader must come from the same version.
Generated upstream files stay out of Git. TGo extensions remain tracked.

See [the preparation instructions](../../../internal/checker/upstream.md) for
file locations and the version-update process.

## Recorded performance measurements

These measurements describe earlier commits, not a guarantee for the current
compiler. They use warm Go build and module caches. Allocations exclude child
Go processes.

The direct-checker comparison used Go 1.27.1 on Linux amd64, with three
compilations per case. The baseline was commit `8d80a62`:

| Source | Helper-based checking | Direct checking | Allocated bytes before / after |
| --- | ---: | ---: | ---: |
| 1 operator | 35 ms | 25 ms | 550 KB / 403 KB |
| 8 independent operators | 133 ms | 25 ms | 2.33 MB / 567 KB |
| 32 independent operators | 400 ms | 29 ms | 15.9 MB / 1.48 MB |
| 32 nested operators | 419 ms | 29 ms | 16.4 MB / 1.44 MB |
| Two packages and a test file, 25 operators | 13.2 s | 398 ms | 3.71 GB / 191 MB |

The single-file cases previously needed 3, 10, and 34 package loads. Direct
checking needs two. Final checking retains types and errors without retaining
syntax information that no later step needs.

A second comparison used copy-based commit `8ba19b9` and disk-stage commit
`e062dfe`, Go 1.27.1, Linux amd64, and `GOMAXPROCS=4`. These are medians of three
one-call samples after a warm-up:

| Input | Module copy | Disk stages | Allocated bytes copy / stages |
| --- | ---: | ---: | ---: |
| 32 propagation calls | 35 ms | 31 ms | 1.47 MB / 1.50 MB |
| Two packages and a test file | 684 ms | 415 ms | 191 MB / 191 MB |
| If value with propagation, no asset | 39 ms | 29 ms | 427 KB / 414 KB |
| Same source with a 32 MiB asset | 164 ms | 135 ms | 67.5 MB / 67.5 MB |

That disk-stage version still read and returned assets. The current driver
lists their paths without reading unrelated asset contents. The
`BenchmarkCompileAssets` benchmark retains this comparison workload.

Frontend parsing has a separate cost. `BenchmarkConditionalParsing` measured
Go 1.27.1 on an AMD EPYC-Turin:

| Input | Earlier preparation | Batched preparation |
| --- | --- | --- |
| 32 independent values | 2.54 ms; 2.56 MB | 0.337 ms; 0.241 MB |
| 512 independent values | 574 ms; 577 MB | 5.14 ms; 4.06 MB |
| 512 nested values | 804 ms; 545 MB | 888 ms; 621 MB |

Independent values benefit from shared edits and parsing. Deep nesting still
needs repeated preparation and subtree parsing; its worst-case cost is quadratic.
This parser work is different from reloading the module for each operator.
