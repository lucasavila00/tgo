# Module compilation

The driver implements `Compile(directory)`. It finds the enclosing Go module,
copies its files into a work directory, and prepares selected TGo sources for
Go parsing. Go's package loader supplies build selection, syntax, and types.

The driver calls `internal/frontend` to record postfix operators and introduce
success-only helper calls. Repeated package loading resolves nested calls.
It checks final error results, forbidden unsafe operands, and whether the
containing function can return each propagated error.

It then passes the typed package and propagation records to
`internal/lowering`. The returned Go replaces helper calls with actual calls,
checks, and returns. Physical source paths control file selection even when
line directives change diagnostic locations.

The result is a map of module-relative file names to contents, including
source and assets. For example, `name_test.tgo` adds `name_tgo_test.go`.
[The architecture](../../docs/arch/transpiler/readme.md) explains the compilation
passes and package dependencies.
