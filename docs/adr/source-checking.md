# Direct TGo source checking

## Decision and reason

Build the private checker and export-data reader from pinned Go source and
tracked patches.
Build matching AST, parser, scanner, printer, and formatter packages from the
same Go pin and tracked patches. The parser creates native TGo nodes. Checking
records types, lexical scopes, and jump targets on those source nodes.

Direct checking resolves `outer(inner()!)!` in one source check. Value branches
need their surrounding types, source scopes, and jump targets. Native nodes
keep those facts on the source expressions that checking and lowering share.

## Cost and boundaries

Maintain a patch against pinned Go source. Go updates require patch review and
validation. Keep private type objects separate from stock objects in public
`Load`. The lowerer preserves execution; stock Go checks the final output.

See the [lowering decision](expression-lowering.md) for statement placement,
the [checker package](../../internal/checker/readme.md) for the current process,
[preparation](../../internal/checker/upstream.md) for Go updates, and
[measurements](../performance.md) for recorded costs.

## Validation

Check nested and independent propagation, multiple results, error-only calls,
error identity and wrapping, concrete errors, type parameters, and return
assignability. Check if-value inference, constants, nil, scopes, and jumps.
Keep public Go type-object identity unchanged.

Check mixed packages, imports, build selection, test variants, stale snapshots,
and original diagnostic positions. Compare behavior with handwritten Go
references as described in the [test guide](../../tests/readme.md).

Require one check per package ID in a compilation and no package reload per
operator. Keep test variants separate. After a Go update, check ordinary Go
acceptance and rejection as well as TGo cases. Use hosted CI for full validation.
