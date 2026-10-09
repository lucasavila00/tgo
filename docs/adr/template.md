# ADR template

Use this template for one architectural decision. Delete all instruction text
when you create the ADR.

Keep the ADR small:

- Target 30 to 80 lines.
- If the ADR exceeds 100 lines, remove detail or split the decision.
- State one decision. Do not combine independent proposals.
- Show the exact public API or data shape when it is the decision.
- Include implementation detail only when the implementation structure is the
  architectural decision.
- Do not add speculative edge cases, tutorials, repeated rationale, or a long
  catalog of alternatives.
- Use short examples instead of long explanations.
- Do not start implementation before approval when the ADR is a proposal.

Use only the sections that help the decision.

```markdown
# <Decision title>

Status: proposed

## Context

<In two to five sentences, state the current problem and why a decision is
needed. Do not summarize the full project.>

## Decision

<State the decision directly. Show the exact API, wire shape, or component
boundary when applicable. Include one or two short examples.>

## Consequences

<List only material compatibility costs, operational effects, or constraints.>

## Alternatives

<Optional. Include only serious alternatives that affect approval. Give each
alternative one short paragraph.>

## Implementation status

<State whether implementation is deferred, active, or complete.>
```

Before review, remove empty optional sections and verify that each remaining
paragraph helps a reviewer approve or reject the decision.
