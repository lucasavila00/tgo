# Feature documentation

This inventory maps each implemented user feature to its short README entry and
its detailed source of truth.

- **TGo source, generated Go, and build behavior.** README: Build and check.
  Detail: [source files and packages](../spec/README.md#source-files-and-packages).
- **Ordinary Go source identity.** README: Build and check. Detail:
  [source files and packages](../spec/README.md#source-files-and-packages).
- **Successful and failure return commas.** README: Return success or failure.
  Detail: [successful returns](../spec/README.md#successful-returns) and
  [failure returns](../spec/README.md#failure-returns).
- **Error propagation with `!` and `!!`.** README: Propagate errors with
  context. Detail: [error propagation](../spec/README.md#error-propagation).
- **Non-nil pointer syntax and proofs.** README: Require non-nil pointers.
  Detail: [non-nil pointers](../spec/README.md#non-nil-pointers).
- **Enum construction, Go ABI, and tag switches.** README: Model closed choices
  and Use TGo packages from Go. Detail: [enum types](../spec/README.md#enum-types)
  and [tag switches](../spec/README.md#enum-tag-switches).
- **Enum JSON and gob forms.** README: Model closed choices. Detail:
  [enum JSON](../spec/README.md#enum-json) and
  [payload-free enum gob](../spec/README.md#payload-free-enum-gob).
- **Checked structs and private fields.** README: Validate construction.
  Detail: [checked structs](../spec/README.md#checked-structs).
- **Field defaults.** README: Declare field defaults.
  Detail: [field defaults](../spec/README.md#field-defaults).
- **Explicit initialization and complete literals.** README: Keep values valid.
  Detail: [explicit initialization](../spec/README.md#explicit-initialization).
- **Collection comprehensions.** README: Build collections.
  Detail: [collection comprehensions](../spec/README.md#collection-comprehensions).
- **Invalid zero values.** README: Keep values valid.
  Detail: [zero validity](../spec/README.md#zero-validity).
- **Map, channel, and assertion presence.** README: Keep values valid.
  Detail: [missing collection values](../spec/README.md#missing-collection-values).
- **Safe reslicing and named results.** README: Keep values valid.
  Detail: [reslicing](../spec/README.md#reslicing) and
  [named results](../spec/README.md#named-results).
- **TGo tests and target selection.** README: Use TGo packages from Go.
  Detail: [source files and packages](../spec/README.md#source-files-and-packages).
- **Go interoperability.** README: Use TGo packages from Go.
  Detail: [Go boundary](../spec/README.md#go-boundary) and the
  [Go caller guide](../guide/GO-CALLERS.md).
- **Reserved generated names and collision diagnostics.** README: Build and
  check. Detail: [reserved generated names](../spec/README.md#reserved-generated-names).
- **Public syntax tree.** README: Tools.
  Detail: [source syntax API](../spec/README.md#source-syntax-api).
- **`tgofmt` source identity, comments, and file modes.** README: Tools. Detail:
  [`tgofmt` guide](../guide/TGOFMT.md).
- **`tgolint` source and caller checks.** README: Keep values valid and Tools.
  Detail: [`tgolint` specification](../spec/TGOLINT.md).
- **`tgolint` diagnostic suppression.** README: Tools. Detail:
  [suppression rules](../spec/TGOLINT.md#suppression).
- **Navigation and hover.** README: Tools.
  Detail: [VS Code guide](../guide/VSCODE.md).
- **VS Code syntax and generated-file hiding.** README: Tools.
  Detail: [extension README](../../editors/vscode/README.md).
- **Generated-output integrity and transactions.** README: Build and check.
  Detail: [build diagnostics](../spec/README.md#build-command-and-diagnostics).

## Update procedure

For each user-visible feature change:

1. Update the normative specification.
2. Add or update the short README entry. Link to detail instead of copying it.
3. Update the user guide, Go caller guide, or tool guide that owns the workflow.
4. Update `docs/for-agents/AGENTS.md` when agent instructions change.
5. Update command comments, package comments, analyzer help, and editor metadata
   when the feature changes their stated scope.
6. Update editor grammar and fixtures when tokens or syntax change.
7. Add or update tests for examples, diagnostics, generated output, and public
   protocol behavior.

For a removed feature, search the README, specification, guides, agent rules,
editor files, examples, fixtures, and generated test data for its old syntax or
name. Keep a rejection test when users can otherwise mistake the removed form
for valid syntax.

Before review, run `make markdown` and check each local link. Run the focused
tool or language tests for the changed documentation. Full validation runs in
hosted CI.
