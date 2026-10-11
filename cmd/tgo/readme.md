# TGo command

This package provides the `tgo` command. It reads one source-directory argument
and calls `pkg/transpiler.Compile`. The directory must belong to a Go module.
The compiler returns file contents keyed by paths relative to that module.

Run `go run ./cmd/tgo /path/to/module` from the repository root. The command
writes generated Go beside the TGo input. For example, `main.tgo` produces
`main_tgo.go`. Test and platform suffixes remain at the end of the name.

Use `-o /tmp/generated` to write the emitted module to another directory. This
mode writes module files, source, and assets. In-place mode writes only Go files
and skips files whose contents are unchanged. The command creates parent
directories before writing. Invalid arguments, compiler errors, and write
errors cause a panic. See the [user guide](../../docs/guide/transpiler.md) for
output rules and examples.

See the [generated Go reference](../../docs/reference/cmd/tgo.md) for exported
declarations and source comments.
