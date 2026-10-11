# Research hypothesis

## Programming languages and review

Older programming languages were too tailored to human editing and human-ness.
Code review is now the bottleneck in agentic development. Review should guarantee
that the work is right for the business requirements, rather than require scanning
every line.

## Communication and verbosity

Communication can be ambiguous. We are often non-native English speakers
communicating through English, which adds room for misunderstanding. Code can
communicate more precisely and efficiently in some contexts, but it is often too
verbose.

Examples:

- **React:** Small visual features need lots of code.
- **Forms:** Simple form rules need code in several places.
- **Sales totals:** Add sales per customer. Several steps in code; one database query.

We need more efficient ways of communicating: more condensed code, tailored to
the reader.

## Data structures and models

Data structures matter. The right model prevents impossible states and mistakes,
both now and in the future.

## Adoption and automatic modernization

Automatic modernization is a nice-to-have. Design and implement language
features for direct use in TGo. Add linter rules and automatic fixes where
existing code has an equivalent form. Each automatic fix must preserve behavior.

The intended upgrade path is:

1. Select a Go package. Rename all handwritten `.go` source and test files to
   `.tgo`. Keep file suffixes, build constraints, imports, and module paths.
2. Compile the package and correct compiler errors.
3. Run the TGo linter and resolve linter errors. Apply automatic modernization
   fixes where available.

Generated Go files are outputs. Other packages can remain Go. The compiler
checks language validity. Modernization shortens code without changing its
behavior. Linting also blocks bad practices, such as zero-value use in TGo.
Resolve compiler and linter errors to complete the upgrade.

For example, a call followed by `if err != nil { return err }` can become
`call()!!` when both forms have the same behavior. These changes must
preserve types, scope, evaluation order, control flow, error identity, deferred
calls, and public APIs. Use `!` only when the existing error wrapper matches
its behavior. Leave code unchanged when conversion needs a design choice.

## Approaches of interest

- Elixir's approach with DSLs is great.
- TypeScript is more efficient than JSON Schema.
