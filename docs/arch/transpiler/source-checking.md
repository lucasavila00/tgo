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


## Compile without a module copy

Both package loads run in the original module. The first overlay maps each
TGo input to its generated Go path. The private checker reads those TGo bytes
from memory and ordinary Go from disk. The second overlay supplies emitted Go
for the final check. A file excluded by build rules has an ignored overlay,
so its stale snapshot cannot become source. Source files stay unchanged.

Go writes each overlay entry to a temporary backing file and removes it after
loading. Assets and ordinary input files are not copied. Local replacements
remain relative during checking; returned `go.mod` uses absolute replacement
paths so the caller can write output elsewhere.

Warm-cache comparison with copy-based commit `8ba19b9`: Go 1.27.1, Linux amd64,
`GOMAXPROCS=4`, three one-call samples after a warm-up. The table shows medians.
Allocations exclude child processes.

| Input | Copy | Overlay | Allocated bytes copy / overlay |
| --- | ---: | ---: | ---: |
| 32 propagation calls | 35 ms | 41 ms | 1.47 MB / 1.50 MB |
| Two packages and a test file | 684 ms | 641 ms | 191 MB / 191 MB |
| If value with propagation, no asset | 39 ms | 38 ms | 427 KB / 428 KB |
| Same source with a 32 MiB asset | 164 ms | 138 ms | 67.5 MB / 67.6 MB |

This removes asset writes, not the API's asset reads. The overlay test checks
that a 1 MiB asset adds no overlay entry. Thus its temporary payload is zero;
the copy wrote the full asset. Both paths use one metadata load and one final
Go load. Small-input times vary and show no general speed increase.

`BenchmarkCompileAssets` keeps the asset comparison reproducible. Tests check
embed and cgo headers after output is written, excluded snapshots, relative
replacements, unchanged source on failure, and temporary-file cleanup.
