// Package checker checks TGo types, name scope, and jumps before code generation.
//
// It checks the syntax from the [frontend] and supplies types and declarations
// to [lowering]. A successful check means that the source follows TGo's rules.
// The [driver] also checks the generated output with Go's unmodified checker.
//
// # Check TGo expressions in their source context
//
// For a function fetch that returns (Value, error), fetch()! provides a Value
// to the surrounding expression. The checker verifies the error result and the
// enclosing function's return type. It also checks nested uses, such as
// outer(inner()!)!, within the same source package check.
//
// If and switch expressions get their type from their use or branch values.
// For example, this assignment requires both branch values to fit int:
//
//	var count int = if ready { 1 } else { 0 }
//
// Switch cases keep Go scopes, including the types of type-switch bindings.
// A branch that returns from the function does not supply a value. The shared
// [control flow analysis] determines whether a branch can reach its end. Returns,
// defers, and jumps inside added expressions belong to the source function,
// even when the frontend uses temporary function syntax to represent them.
//
// A comprehension declares the slice or map type that it builds. The checker
// uses that type to check each output value. Names from a range clause are
// available to later clauses and the output, but not outside the comprehension.
//
// # Use one compatible set of type information
//
// The standard Go checker does not know TGo's expression rules. This package
// uses a patched copy of it and a matching reader for compiled dependency types.
// Both come from the same pinned Go version. Run make prepare to generate them;
// see [checker maintenance] before changing that version or its patch.
//
// Types from this copy are separate from go/types objects. Keep them separate
// when checking ordinary Go for the public Load API. That API exposes the
// original Go package and its go/types objects to callers.
//
// A loader shares dependency types and completed checks between packages. It
// keeps each test variant separate and uses the target architecture's type
// sizes. Thus, all source checks use the same types and build settings.
//
// [frontend]: https://github.com/lucasavila00/tgo/blob/main/internal/frontend/readme.md
// [lowering]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/readme.md
// [driver]: https://github.com/lucasavila00/tgo/blob/main/internal/driver/readme.md
// [control flow analysis]: https://github.com/lucasavila00/tgo/blob/main/internal/controlflow/readme.md
// [checker maintenance]: https://github.com/lucasavila00/tgo/blob/main/internal/checker/upstream.md
package checker
