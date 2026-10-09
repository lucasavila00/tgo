# Go printer port

Port the Go 1.27.2 printer algorithms to TGo. Keep the `format.Source` API. Replace only node access:
the port reads `pkg/syntax` values and does not import `go/ast`, `go/printer`, or `go/format`.

The source baseline is `$GOROOT/src/go/printer` from Go 1.27.2. Adapted files must keep the Go
copyright header. `third_party/go/LICENSE` contains the required BSD license text.

## Source map

| Go source | Functions to port | TGo dependency |
| --- | --- | --- |
| `printer/math.go` | `log2ish`, `exp2ish` | `math` |
| `printer/comment.go` | `formatDocComment`, `isDirective`, `allStars` | `syntax.Comment`, `go/doc/comment` |
| `printer/gobuild.go` | `fixGoBuildLines`, `appendLines`, `lineAt`, `commentTextAt`, `isNL` | `go/build/constraint`, `slices`, `text/tabwriter` |
| `printer/printer.go` | `commentsHaveNewline` through `flush`, plus `isBlank`, `commonPrefix`, `trimRight`, `stripCommonPrefix`, `nlimit`, `mayCombine`, and `trimmer.Write` | `syntax.CommentGroup`, `go/token`, `io`, `strings`, `text/tabwriter`, `unicode` |
| `printer/nodes.go` common | `linebreak`, `setComment`, `identList`, `exprList`, `parameters`, `combinesWithName`, `isTypeElem`, `signature`, `identListSize`, `isOneLineFieldList`, `setLineComment`, `fieldList` | syntax fields and expression tags |
| `printer/nodes.go` expressions | `walkBinary`, `cutoff`, `diffPrec`, `reduceDepth`, `binaryExpr`, `isBinary`, `expr1`, `normalizedNumber`, `possibleSelectorExpr`, `selectorExpr`, `expr0`, `expr` | `syntax.Expression` |
| `printer/nodes.go` statements | `stmtList`, `block`, `isTypeName`, `stripParens`, `stripParensAlways`, `controlClause`, `isCompositeLitLike`, `indentList`, `stmt` | `syntax.Statement` |
| `printer/nodes.go` declarations | `keepTypeColumn`, `valueSpec`, `sanitizeImportPath`, `spec`, `genDecl`, `sizeCounter.Write`, `nodeSize`, `numLines`, `bodySize`, `funcBody`, `distanceFrom`, `funcDecl`, `decl`, `declToken`, `declList`, `file` | `syntax.Specification`, `syntax.Declaration`, `syntax.File` |

Do not port the public `go/printer` API, `CommentedNode`, or `printNode`. TGo formats one parsed
`syntax.File`. Do not port the `go/ast` node cache. Use separate pointer-keyed size maps for
expressions, statements, specifications, declarations, fields, and field lists.

## Syntax mapping

- Use `Span.Start` and `Span.Stop` for `Pos` and `End`.
- Replace Go type switches with exhaustive switches on `Expression.Tag()`, `Statement.Tag()`,
  `Specification.Tag()`, and `Declaration.Tag()`.
- Use `syntax.Walk` when the Go source uses `ast.Inspect`.
- Use `CommentGroup.List`, `Doc`, and `Comment` directly. These fields match the Go tree shape.
- Map `ast.SEND`, `ast.RECV`, and their combination to `syntax.ChannelDirection` tags.
- Keep `go/token`. It defines the operator and position contracts used by both trees.

Add TGo syntax only at the corresponding printer dispatch:

| TGo node | Printer extension |
| --- | --- |
| `DefaultExpression` | Print `..default`. |
| `PropagationExpression` | Print the call followed by `!` or `!!`. |
| `ComprehensionExpression` | Print its result and clauses with the Go list and block spacing functions. |
| `ReturnStatement` | Print `FailureCommas` before results and `SuccessComma` after results. |
| `CaseClause.Exhaustive` | Print `exhaustive:` through the case-clause path. |
| `EnumDeclaration` | Print variants through the declaration, field, and comment functions. |
| `StructDeclaration` | Print TGo fields, defaults, and the final `checked` token. |

## Stages

1. Keep the exact final tabwriter and trimmer pass. `go_printer_layout.tgo` is this first compiling
   slice. Its tests cover columns, formfeed section boundaries, Unicode width, escapes, and trailing
   space removal.
2. Port printer state, token output, positions, comments, build constraints, and numeric helpers.
   Test the raw event stream against fixed Go 1.27.2 outputs. Do not add node printing yet.
3. Port fields, expression lists, and all Go expression tags. Test ordinary Go expression files
   against `gofmt` and test TGo expression fixtures against checked goldens.
4. Port statements. Then port specifications, declarations, and files. Keep each stage behind an
   internal entry point until its Go corpus slice has byte identity and second-pass identity.
5. Add the seven TGo dispatch cases in the table. Switch `Source` to the port after the fast Go
   corpus, the full hosted Go corpus, and the repository TGo corpus pass.
6. Remove the current formatter implementation and its alignment planners after the new path is the
   only `Source` path. Keep the boundary check that rejects `go/format`, `go/printer`, and external
   `gofmt` use.

Each stage must commit generated `*_tgo.go` files. Run the full Go source corpus in hosted slow CI.
Use the focused formatter tests and fast corpus during local work.
