// Package lowering turns checked TGo expressions into ordinary Go code.
//
// The goal is to replace TGo syntax while keeping the source program's behavior.
// A rule can need several Go statements for one source expression. For example,
// use(first(), fetch()!, last()) must save first, check fetch, and return on
// failure before last runs. Lowering keeps these statements with their values
// until the enclosing operation places them where they execute.
//
// Each [rule package] owns a language decision: error propagation, collection
// construction, or branch results. [Shared operations] own evaluation order,
// typed storage, and Go control flow. Rules use those operations so the same
// source operation follows the same execution rules in every feature.
//
// Checked source facts and selected operations are complete before rewriting
// starts. Rewriting creates its own state and copies changed syntax. Each file
// owns its imports and binding changes. Each function owns its returns and
// jumps. Nested rules use the current function; source closures get a new one.
// This ownership keeps a nested error return in the correct source function.
//
// [rule package]: https://github.com/lucasavila00/tgo/tree/main/internal/lowering
// [Shared operations]: https://github.com/lucasavila00/tgo/blob/main/internal/lowering/shared/readme.md
package lowering
