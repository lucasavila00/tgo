# Source checking

`outer(inner()!)!` needs the inner call's success type before the outer call
can be checked. The private Go checker handles both operators in the same
package check. It checks the final error result and the enclosing function's
error return type. Lowering then uses these checked types.

The driver loads build metadata once and caches source packages by package ID.
A normal package and its test variant have different IDs. After lowering, one
stock Go package load checks the emitted module. The number of package loads
does not grow with the number of operators.

`make prepare` generates private Go sources from the pinned version and tracked
patch. The patch changes import paths, adds the operation map, and calls the
TGo rule from expression checking. The matching import reader keeps imported
objects in the same private type system. Public `Load` keeps stock Go objects.
The loader reads Go's target architecture once, including saved Go configuration,
and uses its word sizes for all source checks.

## Measured cost

Warm Go build and module caches; Go 1.27.1, Linux amd64, three compilations
per case. Times are measured complete `Compile` calls. Allocations are in the
compiler process; they exclude child Go processes.

| Source | Before | After | Allocated bytes before / after |
| --- | ---: | ---: | ---: |
| 1 operator | 35 ms | 25 ms | 550 KB / 403 KB |
| 8 independent operators | 133 ms | 25 ms | 2.33 MB / 567 KB |
| 32 independent operators | 400 ms | 29 ms | 15.9 MB / 1.48 MB |
| 32 nested operators | 419 ms | 29 ms | 16.4 MB / 1.44 MB |
| Two packages and a test file, 25 operators | 13.2 s | 398 ms | 3.71 GB / 191 MB |

The single-file cases previously used 3, 10, and 34 package loads. They now use
2. Final checking requests types and errors, without retaining syntax or
expression information that no later stage uses. The checker cache test removes the source file after the first check and
checks the same package ID again. It also checks that test variants keep
separate type objects. Another test checks that public Go bindings stay intact.

Run the benchmarks with:

```sh
make prepare
go test ./internal/driver -run '^$' -bench 'BenchmarkCompile.*Propagations' -benchtime=3x
```

The baseline is main commit `8d80a62` with the same benchmark file. These results
show this workload's cost; future changes can use the same cases for comparison.


## Compile in the source module

The driver first writes original TGo at its generated Go paths. Go selects
packages from these files, and the private checker checks their TGo meaning.
Lowering replaces selected generated files with ordinary Go. The driver removes
excluded generated paths before stock Go checks the final files on disk.

Only paths owned by TGo inputs in the selected module are compiler outputs.
Data directories, nested modules, and vendor dependencies retain their inputs
and snapshots as assets. Go excludes `testdata` and directories whose names
start with `.` or `_` from package discovery. Ordinary Go and assets
stay at their original paths. Go uses normal module loading and can update
module metadata. Returned output includes current sums and makes local
replacement paths absolute so it can be written elsewhere.

Warm-cache comparison with copy-based commit `8ba19b9`: Go 1.27.1, Linux amd64,
`GOMAXPROCS=4`, three one-call samples after a warm-up. The table shows medians.
Allocations exclude child processes.

| Input | Module copy | Disk stages | Allocated bytes copy / stages |
| --- | ---: | ---: | ---: |
| 32 propagation calls | 35 ms | 31 ms | 1.47 MB / 1.50 MB |
| Two packages and a test file | 684 ms | 415 ms | 191 MB / 191 MB |
| If value with propagation, no asset | 39 ms | 29 ms | 427 KB / 414 KB |
| Same source with a 32 MiB asset | 164 ms | 135 ms | 67.5 MB / 67.5 MB |

Assets add no compiler write step. The API still reads and returns their bytes,
so allocation cost remains. Both paths use one metadata load and one final Go
load. These short samples describe this workload; they do not establish a
speed guarantee. `BenchmarkCompileAssets` retains the asset comparison.

Fixture checks compile temporary copies and read committed snapshots before
compilation. The compiler itself works in place. Tests compare final disk files
with returned bytes, exercise failure followed by correction, and check embed,
cgo headers, vendor selection, workspace dependencies, and module updates.
