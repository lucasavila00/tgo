// Package lowering turns checked TGo expressions into ordinary Go code.
//
// TGo lets a call return an error from its enclosing function and lets branches
// and loops produce values inside expressions. Go needs statements for these
// operations. Lowering puts those statements where the source expression runs,
// using checked types as input. It copies changed syntax so one checked source
// can support independent rewrites. The driver writes the formatted output.
//
// The dispatcher selects [propagation], [comprehensions], or [value expressions].
// Each rule owns its TGo transformation. The [shared operations] handle evaluation,
// typed value storage, assignments, and ordinary Go control flow. This boundary
// lets a rule use the same ordering and storage decisions as other rules.
// The [source inventory] retains checked input facts; the [name allocator]
// reserves source names and supplies fresh generated names.
//
// For use(first(), fetch()!, last()), shared evaluation saves first before the
// propagation rule checks fetch. A failure returns before last or use runs.
// The rule chooses the error action; shared operations preserve the evaluation
// point and the enclosing function's return target.
//
// A context is the state used while rewriting one function. Nested functions
// get their own return and jump state, while sharing file imports and names.
// Rules can contain other rules. Shared evaluation calls back to this
// dispatcher with the current context instead of importing rule packages.
// This keeps package dependencies in one direction and preserves function
// ownership during recursive rewrites. See [shared operations] for the local
// branch callback used by value switches.
//
// [propagation]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/propagation/readme.md
// [comprehensions]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/comprehensions/readme.md
// [value expressions]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/valueexpr/readme.md
// [shared operations]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/shared/readme.md
// [source inventory]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/source/readme.md
// [name allocator]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/names/readme.md
package lowering
