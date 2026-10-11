# Upstream source

Run `make prepare` before building or testing the compiler. The script reads
the Go version from `patches/go-version`, gets that toolchain's source, and
applies `patches/source.patch`. It generates the private checker, matching
compiled-package reader, and their required internal packages. Git ignores
these generated files. The Go license is in `LICENSE`.

The patch contains all upstream changes, including private import paths and
TGo checker hooks. TGo expression code and tests stay in `types/tgo.go` and
`types/tgo_test.go`. The script omits upstream tests, generation entry points,
and public compatibility link names.

To change the Go version, update `patches/go-version` and check the patch
against that source. The script applies patches with zero fuzz. Run focused
checker tests and the full compiler suite after an update.
