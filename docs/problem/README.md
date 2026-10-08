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

## Approaches of interest

- Elixir's approach with DSLs is great.
- TypeScript is more efficient than JSON Schema.
