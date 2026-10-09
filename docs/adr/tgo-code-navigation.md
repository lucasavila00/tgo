# Read-only TGo code navigation

## Decision

Build a VS Code extension for reading `.tgo` files. It provides syntax highlighting,
go to definition, find references, document symbols, and workspace symbols.

It does not provide completion, rename, formatting, code actions, or diagnostics.

Use a TextMate grammar and direct VS Code providers. The providers call a native helper
that is written in `.tgo` and compiled through generated Go. The helper uses the
existing compiler analysis and syntax model.

This design does not copy the TGo parser. Add an LSP adapter only when a second editor
needs the same operations.

## Architecture

```text
.tgo file -> TextMate grammar -> token colors
          -> VS Code provider -> TGo helper -> compiler analysis
          <- exact file and range <- navigation index
```

The helper runs in the VS Code workspace environment. It accepts a file URI and a byte
offset. It returns file URIs and byte ranges. The extension converts these ranges to VS
Code UTF-16 positions.

The helper uses `pkg/syntax`, `compiler.AnalyzePackage`, and source facts. It resolves
symbols by type object identity. It returns no result when it cannot find an exact
range. It does not guess by name.

The package index follows the active Go build configuration. It invalidates an affected
package after a `.tgo`, `.go`, `go.mod`, or `go.work` change.

## Required behavior

- Document and workspace symbols include package declarations, types, functions,
  methods, fields, enum variants, constants, and variables.
- Definition navigation supports identifiers and selectors in workspace files.
- A generated member resolves to its owning TGo type, field, or variant.
- References use resolved object identity across analyzed workspace packages.
- Highlighting covers `%`, `!`, `!!`, `enum`, `where`, `exhaustive`, defaults, and
  comprehensions.
- The first release can require a matching `tgonav` executable on `PATH`.
- The first release does not navigate into downloaded dependencies.

## Verification

Use end-to-end workspace fixtures for every provider. Tests must start the compiled
helper and exercise the same JSON protocol as the extension. Cover:

- definitions and references across files and packages;
- local scope, shadowing, methods, selectors, fields, and enum variants;
- generated member ownership;
- document and workspace symbol kinds and ranges;
- active and excluded build files;
- file changes and index invalidation;
- non-ASCII text and UTF-16 conversion;
- paths that contain spaces; and
- all TGo syntax tokens in the grammar.

Run VS Code integration tests for provider registration, helper startup, cancellation,
and result conversion. Keep semantic assertions in helper tests so they do not depend
on VS Code internals.

VS Code documents the required [syntax](https://code.visualstudio.com/api/language-extensions/syntax-highlight-guide),
[provider](https://code.visualstudio.com/api/language-extensions/programmatic-language-features),
and [extension host](https://code.visualstudio.com/api/advanced-topics/extension-host)
APIs. Its positions use [UTF-16 offsets](https://code.visualstudio.com/api/references/vscode-api#Position).
