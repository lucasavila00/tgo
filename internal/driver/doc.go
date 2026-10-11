// Package driver owns module compilation.
//
// A module is the unit of compilation because imports, build constraints, and
// test packages decide which files form a program. The driver keeps those
// choices consistent across frontend checking and lowering. It uses Go's build
// rules so TGo files belong to the same packages as their Go equivalents.
//
// Source files are authoritative. Existing generated files can be stale and
// must not affect source checking. Compiler work uses a separate directory so
// compilation does not edit the caller's files.
//
// The driver returns the emitted module as file contents. Writing those files
// belongs to the caller. Expression evaluation rules belong to the lowerer.
package driver
