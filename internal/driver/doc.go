// Package driver runs module compilation and writes generated Go.
//
// Compile finds the Go module for the supplied directory. It lists module paths,
// reads owned Go and TGo source, and excludes stale generated files. Assets,
// vendor dependencies, data directories, and nested modules are not source inputs
// for this driver.
//
// The driver writes original TGo at each final generated Go path. Go's package
// tools then select files, imports, build constraints, and test variants. The
// private checker reads the selected source through the frontend. After all
// source checks pass, the lowerer builds ordinary Go and the driver writes it at
// the same generated paths. Excluded generated paths are removed. Stock Go checks
// the final module.
//
// Compilation returns no file contents and creates no temporary module or
// separate temporary source files. Original TGo, ordinary Go, and assets stay at
// their paths. A failure can leave a source version or completed output at a
// generated path. The next compilation replaces it from the TGo input.
//
// Start with compile.go for the steps. module.go handles ownership and disk
// writes. propagation.go coordinates selection, checking, and lowering for both
// propagation and if values. filename.go preserves test and platform suffixes.
// The unsafe operand check rejects propagation inside unsafe size and offset
// operands. Expression behavior belongs to the lowerer.
package driver
