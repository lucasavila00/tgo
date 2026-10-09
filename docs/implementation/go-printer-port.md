# Go printer port

Port the Go 1.27.2 printer algorithm to TGo. Keep `format.Source` and use the public
`pkg/syntax` tree. The formatter must not import `go/ast`, `go/printer`, or `go/format`.

The source baseline is `$GOROOT/src/go/printer`. Adapted files must keep the Go copyright
header. `third_party/go/LICENSE` contains the BSD license text.

## Source map

Port these parts of the Go printer:

- `math.go`: size-ratio helpers.
- `comment.go`: document comment and directive formatting.
- `gobuild.go`: build constraint placement.
- `printer.go`: printer state, positions, comments, whitespace, tokens, and the final tabwriter pass.
- `nodes.go`: fields, lists, expressions, statements, declarations, and files.

Do not port the public `go/printer` API, `CommentedNode`, or `printNode`. TGo formats one parsed
`syntax.File`. Use pointer-keyed size caches for each TGo node category.

## Syntax mapping

- Use `Span.Start` and `Span.Stop` for `Pos` and `End`.
- Replace Go type switches with exhaustive TGo tag switches.
- Use `syntax.Walk` where Go uses `ast.Inspect`.
- Read comments from the public TGo tree.
- Map channel directions through `syntax.ChannelDirection`.
- Keep `go/token` for operators and positions.

Add TGo syntax at its matching dispatch point:

- expressions: `..default`, `!`, `!!`, and comprehensions;
- returns: leading failure commas and the final success comma;
- switches: `exhaustive:`;
- declarations: enums, field defaults, and checked structs.

## Cutover

1. Port the final tabwriter and whitespace trimmer.
2. Port printer state, token output, positions, comments, build constraints, and size helpers.
3. Port fields, lists, and every Go expression tag.
4. Port statements, specifications, declarations, and files.
5. Add the TGo dispatch cases.
6. Switch `Source` after Go and TGo corpus tests pass, then remove the old printer.

Keep each stage internal until its focused byte-parity and second-pass tests pass. Commit generated
`*_tgo.go` files with each stage. Run the full Go corpus only in hosted Slow CI.
