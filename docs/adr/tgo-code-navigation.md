# Read-only TGo code navigation

## Decision goal

TGo needs a small tool for code reading. The first tool must support:

- syntax highlighting for `.tgo` files;
- document and workspace symbols;
- go to definition, including F12 in VS Code; and
- find references.

It will not provide completion, formatting, rename, code actions, diagnostics, or other
editing features.

Syntax highlighting cannot provide semantic navigation. A TextMate grammar classifies
text with regular expressions. It does not resolve imports, scopes, shadowed names,
selectors, or generated model members. VS Code supplies separate APIs for syntax
highlighting and programmatic language features such as definition and references
([syntax guide](https://code.visualstudio.com/api/language-extensions/syntax-highlight-guide),
[language feature guide](https://code.visualstudio.com/api/language-extensions/programmatic-language-features)).

## Repository facts

The repository already has most of the semantic base:

- [`pkg/syntax`](../../pkg/syntax/ast.tgo) defines a read-only TGo tree with source
  spans.
- [`ParseFile`](../../pkg/syntax/convert.tgo), `Walk`, `Inspect`, `Children`, `Parent`,
  `ExtensionAt`, and the query helpers give a complete public syntax interface.
- [`compiler.AnalyzePackage`](../../internal/compiler/analysis.go) checks one package
  without file writes. It returns each TGo tree, generated output, the checked Go
  package, non-nil facts, and an internal source-fact index.
- [`sourcefacts.Index`](../../internal/sourcefacts/index.tgo) maps TGo identifiers and
  expressions to `go/types` objects, types, selections, and generic instances. This
  package is internal and does not yet expose a navigation result model.
- Each generated file has a `//tgo:v2 "source.tgo"` owner line. The compiler uses
  temporary `//line` mappings during checking, then removes them before it writes the
  `*_tgo.go` file. Go defines `//line` as a way to change the reported position of the
  following code ([Go documentation](https://go.dev/wiki/Comments#compiler-directives)).

These facts imply that a TGo helper can use the compiler's checked projection and
return source locations. They do not imply that `gopls` can navigate `.tgo` files.
The written Go file names its owner, but it has no complete source map.

## Options

| Option | Highlighting | Navigation | Main cost or limit |
| --- | --- | --- | --- |
| VS Code and helper | TextMate | Direct providers | One editor; TGo helper required. |
| TGo language server | Editor grammar | LSP methods | More protocol and process state. |
| Tree-sitter grammar | Highlight queries | Tags and helper | Duplicates the TGo parser. |
| Monaco web browser | Monarch | Monaco providers | Needs a service, hosting, and repository access. |
| Generated Go and `gopls` | Go grammar | Result mapping | No exact persistent source map. |

VS Code extensions can register providers directly. They do not need an LSP server.
The official API lists direct definition, reference, document symbol, and workspace
symbol providers
([VS Code guide](https://code.visualstudio.com/api/language-extensions/programmatic-language-features)).
An LSP server becomes useful when several editors must share the same service. LSP
standardizes JSON-RPC between editors and language servers
([LSP overview](https://microsoft.github.io/language-server-protocol/)).

Tree-sitter can parse during each edit and can emit syntax tags for definitions and
references. Its navigation guide describes these as parts of a code navigation system,
not as a type checker
([Tree-sitter introduction](https://tree-sitter.github.io/tree-sitter/),
[code navigation](https://tree-sitter.github.io/tree-sitter/4-code-navigation.html)).
Monaco also has a definition provider, but the host must supply its result
([Monaco API](https://microsoft.github.io/monaco-editor/typedoc/modules/editor_editor_api.languages.html)).

## Decision

Start with a desktop and remote-workspace VS Code extension. Use a TextMate grammar for
color and direct VS Code providers for navigation. Run a small native helper written in
`.tgo`. Compile it through generated Go into an ordinary executable. The helper can
import and use the existing Go compiler and syntax packages.

This is the smallest sound path to highlighting and F12. The extension stays thin, and
the helper does not copy compiler rules. It also makes the navigation tool use TGo. The
helper executable must run where the workspace files are. VS Code has local and remote
Node.js extension hosts for this model. Browser extension hosts cannot start an executable
([extension hosts](https://code.visualstudio.com/api/advanced-topics/extension-host),
[web extension limits](https://code.visualstudio.com/api/extension-guides/web-extensions)).

The first release can require a matching `tgo` tool on `PATH`. A later release can
bundle one helper for each supported operating system and architecture. Bundling does
not change the protocol or the editor providers.

## Minimal architecture

```text
.tgo document
  -> TextMate grammar -> token colors
  -> VS Code provider -> native TGo helper -> compiler.AnalyzePackage
                                  -> syntax tree + source facts
  <- exact file and range <- navigation index
```

The helper keeps a package index for the active Go build configuration. It invalidates
the affected package when a `.tgo`, `.go`, `go.mod`, or `go.work` file changes. The
extension sends a file URI and a byte offset. The helper returns file URIs and byte
ranges. The extension converts them to VS Code positions. VS Code character offsets
use UTF-16 code units, so it must not treat a Go byte column as a VS Code character
offset ([VS Code `Position`](https://code.visualstudio.com/api/references/vscode-api#Position)).

The helper must return no result when it has no exact range. It must not guess by name.

## Exact first operations

- **Document symbols:** package declarations, types, functions, methods, fields, enum
  variants, constants, and variables in one `.tgo` file.
- **Workspace symbols:** the same declarations in build-selected `.tgo` files in the
  open workspace.
- **Go to definition:** identifiers and selectors whose definition has an exact range
  in a build-selected workspace `.tgo` or `.go` file. A generated model member goes to
  its owning TGo type, field, or variant declaration.
- **Find references:** resolved `go/types` object identity inside one analysis, with a
  stable package and object key across package analyses. The result can include the
  declaration when VS Code requests it.

The first release does not promise navigation into downloaded dependency source. It
also does not return results from files that the active Go build excludes. `gopls` has
the same general build-configuration limit for reference results
([gopls navigation](https://go.dev/gopls/features/navigation#references)).

## Risks and gaps

- `compiler.AnalyzePackage` and `sourcefacts` are internal. The helper must live in this
  repository, or the repository must first expose a small stable navigation API.
- The source-fact index can resolve an object at an identifier, but it does not expose
  all reference locations. The helper must walk identifiers and group uses by the
  resolved object. It also needs a stable key for objects used by other packages.
- Generated members need explicit owner ranges. The generated-file owner line alone is
  insufficient.
- TextMate rules must cover TGo tokens before they include the base Go grammar. They
  need fixtures for `%`, `!`, `!!`, `enum`, `where`, `exhaustive`, defaults, and
  comprehensions.
- Package analysis can be slow on a cold workspace. Cache checked packages and cancel
  obsolete requests. Do not add a second parser cache in the extension.

## Staged work

1. Add a small `.tgo` helper command and protocol. Test symbols, definition, references,
   generated member ownership, build selection, and non-ASCII offsets. Build its
   generated Go as a native executable.
2. Add the VS Code language ID, TextMate grammar, and direct providers. Test local and
   remote workspaces with the helper on `PATH`.
3. Measure use. Add an LSP adapter only when another editor needs the same operations.
   Do not add a Tree-sitter grammar or web browser until one has a separate user need.
