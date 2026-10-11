// Package driver owns module compilation.
//
// A module is the unit of compilation because imports, build constraints, and
// test packages decide which files form a program. The driver keeps those
// choices consistent across frontend checking and lowering. It uses Go's build
// rules so TGo files belong to the same packages as their Go equivalents.
//
// Source files are authoritative. Existing generated files can be stale and
// must not affect source checking. Two overlays replace checking contents at original paths without editing
// the caller's files. Ordinary Go and assets stay on disk.
//
// Compilation passes data through these steps: read module files, prepare a
// source overlay, select source with Go's build rules, check source packages,
// lower expressions into the output overlay, check emitted Go, and collect output.
// Each step keeps the data required by the next step.
//
// The driver returns the emitted module as file contents. Writing those files
// belongs to the caller. Expression evaluation rules belong to the lowerer.
package driver
