# Research: TGo dogfooding opportunities

## Purpose

This document finds production code that can use more TGo features. It does not propose a broad
rewrite. A good candidate must meet these conditions:

- It is outside the pure-Go compiler boundary.
- A TGo feature removes an invalid state or a repeated control-flow pattern.
- The generated Go stays reviewable and has a CI integrity check.
- The change tests a real user path. It does not add a sample only to increase feature use.

The priority terms in this document have these meanings:

- P0: do this before other dogfooding work.
- P1: do this after the P0 work.
- P2: do a small trial after the behavior risk is clear.

## Current coverage

The repository already uses TGo in two production areas. The
[`pkg/syntax`](../../pkg/syntax) package has 12 TGo source files, and
[`internal/tgolint`](../../internal/tgolint) has 25. These areas exercise enums, `match`, non-nil
pointers, and error propagation. The linter also uses internal enums for nil state, scalar state,
model state, and generic effect state.

The coverage has two limits:

1. The generated-output check lists only `pkg/syntax` and `internal/tgolint` in
   [`scripts/check_generated.py`](../../scripts/check_generated.py). A new TGo production package
   can have stale generated output without a CI failure.
2. Error propagation has only one production use. The parser still contains many manual error
   branches.

The syntax model now gives required enum payloads and child nodes non-nil types in
[`pkg/syntax/ast.tgo`](../../pkg/syntax/ast.tgo). This is strong dogfooding. The remaining work is
to carry these contracts into consumers where the call graph proves the same invariant.

The compiler boundary is intentional. [`docs/bootstrap.md`](../bootstrap.md) says that the compiler
must stay in pure Go for now. It also permits TGo in tools that do not affect generated program
runtime code. The candidates below follow that rule.

## Strong candidates

### P0: make the generated-output check cover each production TGo root

Evidence: `GENERATED_ROOTS` in
[`scripts/check_generated.py`](../../scripts/check_generated.py) contains two fixed paths. The
script builds the compiler, regenerates those paths in a temporary repository, and compares the
result with committed `*_tgo.go` files. The `generated` target in the
[`Makefile`](../../Makefile) runs this script in CI.

Feature exercised: this is support for all TGo features rather than one language feature. It makes
new production dogfooding safe.

Expected value: a conversion of `internal/sourcefacts` or another package will get the same stale
output protection as the current TGo packages. Reviewers can trust that a committed generated file
comes from its adjacent source and the current compiler.

Constraints and risks:

- Do not scan all `*_tgo.go` files. Test fixtures contain intentional generated output and error
  cases.
- Keep an explicit list of production roots, or add a small production-root manifest.
- Build local TGo imports before their users if a new root depends on another TGo root.

This check is a prerequisite for the next candidate.

### P0: convert `internal/sourcefacts` and add non-nil contracts

Evidence: [`internal/sourcefacts/index.go`](../../internal/sourcefacts/index.go) maps public TGo
syntax nodes to Go type information. Only TGo linter code imports this package. Its `Index` stores
required pointers to a file set and type information. Its methods also require the index receiver.
`New` requires a Go file. The package has no compiler build or emit responsibility.

Feature exercised: rename `index.go` to `index.tgo`, emit `index_tgo.go`, and use `%T` for values
that are required for all valid calls. The first useful set is:

- the `Index.files` and `Index.info` fields;
- the `New` parameters for the Go file, type information, and file set;
- the `Index` receiver on its methods;
- syntax arguments where all callers already prove that the node exists.

Keep optional results nullable. `FunctionSignature`, `CalledFunction`, and the internal expression
lookup can correctly return no value. Do not change those results only to increase `%T` use.

Expected value:

- It tests non-nil facts across three production packages: `pkg/syntax`, `internal/sourcefacts`,
  and `internal/tgolint`.
- It replaces undocumented panic preconditions with checked TGo contracts.
- It tests a mixed package graph instead of another file in an existing TGo package.

Constraints and risks:

- Add `internal/sourcefacts` to the generated-output check in the same change.
- Run `tgolint` on callers. Some syntax helpers accept nil on purpose and must keep `*T`.
- Keep the generated Go API compatible. `%T` emits `*T`, so the Go representation and ABI do not
  change.

### P1: carry syntax non-nil contracts into linter helpers

Evidence: [`pkg/syntax/ast.tgo`](../../pkg/syntax/ast.tgo) now uses non-nil elements in syntax node
lists and non-nil payloads in its enums. Several private helpers in
[`internal/tgolint/source_syntax.tgo`](../../internal/tgolint/source_syntax.tgo) still take nullable
syntax pointers. Examples include `sourceAssignment`, `sourceCall`, `sourceIdentifier`,
`sourceGeneralDeclaration`, and `sourceValueSpecification`. Their main callers read values from
non-nil syntax lists or from matched non-nil payloads.

Feature exercised: use `%syntax.Statement`, `%syntax.Expression`, `%syntax.Declaration`, and
`%syntax.Specification` for helper inputs where every caller has the exported proof.

Expected value:

- It tests that non-nil facts from `pkg/syntax` reach a separate TGo package.
- It removes nil branches that cannot run for parser-built syntax.
- It makes the new syntax invariant visible at the analysis boundary.

Constraints and risks:

- Audit each caller. Some public query helpers accept nil on purpose and must keep nullable inputs.
- Keep optional return values nullable. A failed shape query must still return `nil, false`.
- Do not change recovery behavior for malformed syntax trees that the public API permits.

### P1: replace the manual `sourceModel` sum type with an enum

Evidence: `sourceModel` in
[`internal/tgolint/source_models.tgo`](../../internal/tgolint/source_models.tgo) has `fields`,
`base`, and `variants` in one struct. `sourceDeclaration` uses only one of these shapes:

- a checked type uses `base` and a model fact;
- an enum uses `variants` and a model fact;
- a struct uses `fields` and no model fact.

The other field combinations are invalid. `sourceShapeMatches` recovers the active shape through
`fact == nil`, `modelIsChecked`, and the final enum path.

Feature exercised: define a private enum with `Checked`, `Enum`, and `Struct` payloads. Use an
exhaustive dispatch in `sourceDeclaration`, `sourceShapeMatches`, and fact export.

Expected value: the linter will test a payload-bearing enum in a verification path that processes
real TGo source. The type will make invalid combinations impossible in TGo code. It can also remove
several nil fields and kind tests.

Constraints and risks:

- Keep `modelWireFact` as a plain struct. It is the stable `go/analysis` fact format.
- Keep source order and diagnostic positions unchanged.
- Do not combine this change with unrelated changes to model verification.

### P2: use postfix `!` in selected parser paths

Evidence: production TGo has one postfix `!` use in
[`internal/tgolint/source_models.tgo`](../../internal/tgolint/source_models.tgo). Manual propagation
remains in these areas:

- `makeFields`, `parseField`, and `parseExpression` calls in
  [`pkg/syntax/parser_build.tgo`](../../pkg/syntax/parser_build.tgo);
- `closeToken`, `rawFields`, and declaration parsing in
  [`pkg/syntax/parser.tgo`](../../pkg/syntax/parser.tgo);
- match parsing and source-tree construction in
  [`pkg/syntax/parser_extensions.tgo`](../../pkg/syntax/parser_extensions.tgo).

Feature exercised: postfix `!` with single and multiple non-error results.

Expected value: these paths test the feature in recursive parser control flow. They also reduce
repeated zero-result returns when the generated wrapper is the intended error contract.

Constraints and risks:

- Postfix `!` adds the static call name to the error. Many parser helpers now return exact syntax
  diagnostics without an added wrapper.
- Parser errors are part of the public `ParseFile` behavior and have end-to-end snapshots.
- Convert one call chain first. Accept it only if the new error text is intentional and tests cover
  it. Keep manual branches where they map positions, combine scanner errors, or preserve text.

This is P2 because feature use does not justify an accidental diagnostic change.

## Areas that should remain Go

### Compiler and compiler command

Keep [`internal/compiler`](../../internal/compiler) and
[`cmd/tgo/main.go`](../../cmd/tgo/main.go) in Go. They contain many apparent candidates for enums,
non-nil pointers, and error propagation. A conversion would make the compiler depend on generated
output from itself. That conflicts with the current bootstrap policy.

This rule also covers [`internal/outputname`](../../internal/outputname). The compiler imports it
during output discovery. Its small path rules do not justify a bootstrap exception.

### Generated Go

Keep each `*_tgo.go` file generated. Do not add hand edits to use a faster or clearer Go shape.
Make the change in the adjacent `.tgo` source or in the compiler emitter. The integrity check must
reproduce the committed file.

### Analysis wire facts

Keep `modelWireFact` in
[`internal/tgolint/models.tgo`](../../internal/tgolint/models.tgo) and `genericEffectWireFact` plus
its child wire structs in
[`internal/tgolint/generic_effects.tgo`](../../internal/tgolint/generic_effects.tgo) as plain
serializable structs. External package facts can be absent, old, or invalid. Their decode functions
must validate integer tags before they create internal TGo enums. The current wire-to-enum boundary
is a useful design and already uses TGo correctly.

### Tests and test data

Keep Go tests in `.go` files. The source rules in
[`docs/spec/README.md`](../spec/README.md) exclude `_test.tgo` files. The Go tool runs tests after TGo
emits Go.

Keep TGo files under `testdata` as compiler and linter fixtures. The production build walk ignores
`testdata`, and the test runners invoke these fixtures explicitly. They do not form production
dogfooding roots.

### Build-constrained compiler files

The Unix and non-Unix implementations in
[`internal/compiler/sync_directory_unix.go`](../../internal/compiler/sync_directory_unix.go) and
[`internal/compiler/sync_directory_other.go`](../../internal/compiler/sync_directory_other.go)
would exercise TGo target suffixes and build constraints. They must stay Go because they are in the
compiler.

There is no current non-compiler production file that needs a platform split. Do not add an
artificial platform file for coverage. Continue to test target suffixes and malformed constraints
in the end-to-end fixtures until a real tool requirement appears.

### Thin command wrapper

Keep [`cmd/tgolint/main.go`](../../cmd/tgolint/main.go) in Go for now. It only calls
`multichecker.Main`. A TGo source file there would add generated output but would not exercise a
TGo safety or modeling feature.

## Recommended sequence

1. Extend generated-output verification to each selected production root.
2. Convert `internal/sourcefacts` and add only proved non-nil contracts.
3. Carry proved syntax contracts into private linter helpers.
4. Replace `sourceModel` with a private enum.
5. Trial postfix `!` on one parser call chain with exact diagnostic tests.

Each step must pass `make generated`, `make dogfood`, the end-to-end tests, the linter tests, and the
unit tests. This sequence first improves CI trust, then tests contracts across package boundaries,
then improves internal models and selected error paths.
