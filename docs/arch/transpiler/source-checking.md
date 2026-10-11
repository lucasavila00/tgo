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

## Measured cost

Warm Go build and module caches; Go 1.27.1, Linux amd64, three compilations
per case. Times are measured complete `Compile` calls. Allocations are in the
compiler process; they exclude child Go processes.

| Source | Before | After | Allocated bytes before / after |
| --- | ---: | ---: | ---: |
| 1 operator | 35 ms | 31 ms | 550 KB / 357 KB |
| 8 independent operators | 133 ms | 30 ms | 2.33 MB / 554 KB |
| 32 independent operators | 400 ms | 30 ms | 15.9 MB / 1.54 MB |
| 32 nested operators | 419 ms | 31 ms | 16.4 MB / 1.50 MB |
| Two packages and a test file, 25 operators | 13.2 s | 568 ms | 3.71 GB / 341 MB |

The single-file cases previously used 3, 10, and 34 package loads. They now use
2. The checker cache test removes the source file after the first check and
checks the same package ID again. It also checks that test variants keep
separate type objects. Another test checks that public Go bindings stay intact.

Run the benchmarks with:

```sh
make prepare
go test ./internal/driver -run '^$' -bench 'BenchmarkCompile.*Propagations' -benchtime=3x
```

The baseline is main commit `8d80a62` with the same benchmark file. These results
show this workload's cost; future changes can use the same cases for comparison.
