# Direct TGo source checking

## Decision and reason

Use a private copy of Go's type checker and its matching export-data reader.
Keep Go's standard parser, with frontend preparation for added syntax.

The earlier driver inserted helpers and reloaded packages to discover success
types. Direct checking resolves `outer(inner()!)!` in one source check. If values
also need branch types, surrounding types, scope, and jumps checked together.
Writing a fake Go file remains a possible design, but requires a representation
of those rules. Writing to disk alone does not supply them.

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
