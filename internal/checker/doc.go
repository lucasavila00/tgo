// Package checker decides whether TGo source is valid before code generation.
//
// It checks types, names, and jumps in the source program. For example, fetch()!
// must have a final error result, and the source function must accept that error.
// An if expression checks each completing branch against the required type.
// A branch that returns from the function does not supply a value.
//
// # Keep the source function's rules
//
// The frontend uses temporary calls to represent added syntax. The checker
// reads their records and checks the statements in the enclosing source scope.
// Returns, defers, and jumps keep their source targets. The shared control-flow
// analysis tells checking and lowering which paths can produce a value.
//
// TGo checking uses a patched Go checker. The pinned version and patch make its
// source reproducible; see [checker maintenance]. Its type objects stay separate
// from stock go/types objects. The driver checks generated Go with stock Go.
//
// A loader owns imported types and completed package checks. Test variants have
// separate identities. Target sizes are read before loader construction, so
// the chosen architecture is an explicit input. Each package supplies syntax
// and TGo records together; checking registers the records before type analysis.
// TGo-specific checking state belongs to that package's checker. Go's environment
// owns the current function, scope, and statement context.
//
// [checker maintenance]: https://github.com/lucasavila00/tgo/blob/main/internal/checker/upstream.md
package checker
