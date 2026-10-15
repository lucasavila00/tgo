# Recorded compiler performance

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

Frontend parsing has a separate cost. Before direct grammar parsing,
`BenchmarkConditionalParsing` measured
Go 1.27.1 on an AMD EPYC-Turin:

| Input | Earlier preparation | Batched preparation |
| --- | --- | --- |
| 32 independent values | 2.54 ms; 2.56 MB | 0.337 ms; 0.241 MB |
| 512 independent values | 574 ms; 577 MB | 5.14 ms; 4.06 MB |
| 512 nested values | 804 ms; 545 MB | 888 ms; 621 MB |

The preparation-based parser shared edits for independent values. Deep nesting
repeated preparation and subtree parsing, with quadratic worst-case cost.
The current parser reads all TGo grammar directly in one parse.

The helper-based direct parser used Go 1.27.1, Linux amd64, the same CPU model, and
`GOMAXPROCS=2`. These samples use one iteration per case:

```sh
go test ./internal/frontend -run '^$' -bench '^BenchmarkConditionalParsing$' -benchtime=1x
```

| Input | Direct parser | Allocated bytes |
| --- | ---: | ---: |
| 32 independent values | 0.183 ms | 68.2 KB |
| 512 independent values | 14.7 ms | 5.95 MB |
| 512 nested values | 39.6 ms | 6.52 MB |

That parser searched from name 1 for each helper. Repeated name searches added
work as the number of forms grew. Native AST parsing retains the source nodes
directly and assigns no helper names.

The native AST parser used the same Go version, target, CPU model, command,
and `GOMAXPROCS=2`, also with one iteration per case:

| Input | Native AST parser | Allocated bytes |
| --- | ---: | ---: |
| 32 independent values | 0.100 ms | 28.2 KB |
| 512 independent values | 0.768 ms | 432.5 KB |
| 512 nested values | 12.7 ms | 1.00 MB |

Read the [driver](../internal/driver/readme.md), [checker](../internal/checker/readme.md),
and [frontend](../internal/frontend/readme.md) for the current process.
