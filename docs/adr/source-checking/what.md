# Direct TGo source checking

## Decision

Use a private copy of Go's type checker and a matching compiled-package reader.
Check propagation and if values before generating Go statements. Keep Go's
standard parser, with frontend preparation for added syntax.

## Why

For `outer(inner()!)!`, the outer call needs the inner success type. The earlier
driver inserted helpers and reloaded packages to expose that type. The private
checker can give `inner()!` its success type and continue checking the outer
call in one source check.

If values also need branch types, surrounding types, scopes, and jumps checked
together. A parser change alone cannot supply these type rules.

## Cost and boundaries

The compiler maintains a patch against pinned Go source. Go updates need patch
review and compiler validation. The lowerer still preserves execution, and
stock Go still checks the final output. Public `Load` retains stock Go objects;
private type objects remain separate.

This replaces the stock-checker requirement in the
[expression-lowering plan](../expression-lowering/how.md). See
[HOW](how.md) for the implementation and [PROOF](proof.md) for required checks.
