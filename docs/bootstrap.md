# Bootstrap

The tgo compiler is implemented in Go. A compiler build does not require an existing tgo
compiler.

Repository tools outside the compiler bootstrap path can use tgo. For example, `tgolint` uses
tgo source files.

Each committed `.tgo` source file has a generated `*_tgo.go` file in the same directory. The Go
tool uses these generated files as normal Go source. `make generated` verifies that the committed
files match the current compiler output.
