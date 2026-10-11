# Build the private Go checker

TGo uses a patched copy of Go's type checker and its compiled-package reader.
The copy belongs to the compiler implementation. It does not replace the
installed Go toolchain, and final output is checked with stock Go.

## Prepare source

Run `make prepare` before building the compiler. It calls
`scripts/prepare-checker.sh`, which:

1. Reads `internal/checker/patches/go-version` and finds that toolchain's source.
2. Copies the checker, export reader, and required internal packages into a
   build-preparation directory.
3. Applies `internal/checker/patches/source.patch`. A shifted or approximate
   patch match is an error.
4. Writes generated private source under `internal/checker/`.

This preparation directory is for building the compiler. Module compilation
itself uses original module paths and creates no temporary module.

## Tracked and generated files

The pinned version, patch, preparation script, and TGo extensions are tracked.
Generated upstream Go files are ignored by Git; do not edit them. The checker
and importer must use the same Go version and private type objects.

TGo rules and tests live in `types/tgo.go` and `types/tgo_test.go`. The patch adds
checker hooks and private import paths. Preparation omits upstream tests,
generation entry points, and public compatibility link names. The upstream Go
license is in `LICENSE`.

## Update Go

Change `patches/go-version`, review the patch against that source, and regenerate.
The script rejects both patch fuzz and offset matches. Check ordinary Go types
and imports as well as TGo propagation, if values, scopes, and jumps. Use hosted
CI for full compiler validation. See
[the checker package](readme.md) for the runtime
compilation process.
