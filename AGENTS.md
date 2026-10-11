# Working principles

## Repository status

We are developing version 2 of the TGo compiler, `compilerv2`, through
[#248](https://github.com/lucasavila00/tgo/issues/248) and
[#247](https://github.com/lucasavila00/tgo/issues/247). Only the README syntax
examples and project documentation remain. Code, tests, build tools, CI, and architecture
documents have been removed. Git history retains them.

Build compilerv2 from the [compiler v2 plan](docs/adr/expression-lowering/what.md).
Do not restore the old compiler code or architecture for now. The old
specification, guides, and user agent rules have been removed. Use the README
for syntax examples.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes,
simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:

- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:

- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:

```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it
work") require constant clarification.

## 5. Complete the Requested Work

**Do the full job. Do not replace it with an easier subset.**

- Do not invent urgency, deadlines, or pressure to save time, effort, or tokens. Only the user can set those tradeoffs.
- Preserve the user's intended outcome and full scope. Simplicity means the least complexity that completely solves the problem, not less of the requested problem.
- Do not silently defer difficult requirements, leave required work to the user, or call a partial implementation complete. If a requirement needs more work, do that work.
- Investigate the actual code, data, and constraints. Fix root causes and carry changes through every affected layer needed for the requested behavior.
- Define acceptance criteria from the user's request, then verify the complete result against them. Passing narrow tests does not prove a broader goal is achieved.
- If a real blocker prevents completion, state exactly what is blocked and why. Do not manufacture a blocker or request permission to avoid already authorized work.
- When corrected, update the implementation and verification scope immediately. Acknowledging the correction or rewriting the plan is not completion.

## 6. Use STE100

Always use ASD-STE100 Simplified Technical English in all writing, including
documentation, code comments, commit messages, and responses to the user.

## 7. Never Rewrite Git History

Never change, replace, or remove an existing commit. Do not amend commits,
rebase branches, move branches with `git reset`, or use a force-push. Add a new
commit for each correction. Merge remote changes when branches have diverged.

## 8. Do Not Add Unrequested Mechanisms

Do not add safety checks, runtime behavior, abstractions, or support systems
that the user did not request. Follow each stated trust assumption. Do not add
checks for data or boundaries that the user declared trusted.

## 9. Validate Documentation

Check changed documentation and its local links. No CI or compiler test suite
exists in this checkout. compilerv2 must provide focused validation.

## 10. Do Not Post Comments Without a Request

Do not create a GitHub issue comment, pull request comment, review comment, or
discussion reply unless the user explicitly asks for that comment. Do not post
comments under the user's identity on your own initiative.

This rule does not prevent the creation or update of an issue body or pull
request body when the user requests that issue or pull request.

## 11. Lower Valid TGo Before You Restrict It

Preserve valid TGo semantics, evaluation order, scope, control flow, and error
identity first. Generate fresh locals and blocks when the lowering needs them.
Improve generated-code appearance after correctness is complete.

Reject only invalid or unrepresentable programs. Do not replace a feasible
compiler lowering with a linter restriction. Do not require users to write
compiler bookkeeping by hand.
