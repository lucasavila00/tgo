# Postfix syntax

The frontend prepares TGo error propagation for Go's parser. It uses Go's
scanner and AST utilities. The driver supplies source bytes, reserved names,
and a map in which to record operations.

`MarkPostfix` replaces postfix `!` or `!!` after a call with a unique comment.
For example, `read()!!` becomes a Go call followed by its marker comment.
The operation records the source position and whether the error needs wrapping.

`MarkerExpressions` finds the call associated with each recorded comment.
`WrapMarker` replaces that expression with a helper call.
`MarkerDeclaration` builds a generic helper with the required number of
success results. The helper lets Go check the surrounding expression without
the call's final error result. The driver later sends these operations to the
lowerer, which removes the helpers from output.

See [the architecture](../../docs/arch/transpiler/readme.md) for nested-call
preparation and the checks performed after typing.
