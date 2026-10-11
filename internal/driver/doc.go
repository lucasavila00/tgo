// Package driver owns module compilation.
//
// A module is the unit of compilation because imports, build constraints, and
// test packages decide which files form a program. The driver keeps those
// choices consistent across frontend checking and lowering. It uses Go's build
// rules so TGo files belong to the same packages as their Go equivalents.
//
// Source files are authoritative. Existing generated files can be stale and
// must not affect source checking. The driver first writes original TGo at
// generated paths, then replaces it with checked generated Go.
//
// Compilation passes data through these steps: read module files, prepare a
// generated Go path for each TGo input, select source with Go's build rules,
// check source packages, lower expressions, and check emitted Go.
// Each step keeps the data required by the next step.
//
// The driver writes generated Go in the source module. Expression evaluation
// rules belong to the lowerer.
package driver
