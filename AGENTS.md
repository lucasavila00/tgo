# Working principles

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

## 9. Use Hosted CI for Full Validation

Do not run the full CI suite locally. Push the branch and use GitHub Actions
for full validation. Run focused local tests only to validate the changed area
or debug a CI failure.

## 10. Do Not Post Comments Without a Request

Do not create a GitHub issue comment, pull request comment, review comment, or
discussion reply unless the user explicitly asks for that comment. Do not post
comments under the user's identity on your own initiative.

This rule does not prevent the creation or update of an issue body or pull
request body when the user requests that issue or pull request.

## 11. Lower Valid TGo Before You Restrict It

Read the [compiler lowering guide](docs/contrib/compiler-lowering.md) before
you change compiler lowering or add a source restriction.

Preserve valid TGo semantics, evaluation order, scope, control flow, and error
identity first. Generate fresh locals and blocks when the lowering needs them.
Improve generated-code appearance after correctness is complete.

Reject only invalid or unrepresentable programs. Do not replace a feasible
compiler lowering with a linter restriction. Do not require users to write
compiler bookkeeping by hand.

## 12. Autonomous Work

Move the requested work to review without unnecessary waiting.

- Define the complete acceptance criteria before implementation. Keep the
  active pull request within that scope. Create a separate issue for an
  unrelated finding.
- The root agent coordinates the work, defines the scope, tracks hosted CI,
  and reviews repository state. It must not create or edit repository files.
- Only a `gpt-5.6-sol` subagent can create or edit repository files. Start each
  subagent with the `gpt-5.6-sol` model, and give it the full task context.
- The runtime has four total agent slots. The root agent uses one slot, so
  three subagent slots are available.
- Use slots in short cycles. One subagent implements one issue, runs focused
  checks, pushes a complete draft pull request, reports its exact head commit,
  and then stops.
- The root agent tracks hosted CI and review without keeping the implementation
  subagent active. Immediately use the free slot for the highest-priority
  unblocked issue. Do not wait for the prior pull request to merge.
- Use a separate `gpt-5.6-sol` subagent for the final independent review. The
  review subagent reports its result and then stops.
- Do not use an agent slot to wait for hosted CI.
- Do not merge `main` into a clean pull request only to refresh its ancestry.
  This repository squash-merges pull requests. Merge `main` only when GitHub
  reports a conflict, and preserve both sides of the conflict.
- Do not use a GitLab-style merge train that repeatedly merges the latest
  `main` into every open pull request. Review and test each clean pull request
  at its own head. Run the full tests again on the combined `main` branch
  before publishing.
- Keep a pull request in draft while implementation, conflict resolution,
  review, or required CI is incomplete.
- Mark a pull request ready as soon as its requested scope is complete, its
  independent review has no blocker, GitHub reports no conflict, and required
  hosted checks pass.
- Rebuild the work inventory after a merge or a new issue changes priorities.
  Do not repeat the inventory while the external state is unchanged.
- Keep no more than six pull requests open at one time. Draft pull requests can
  outnumber active agents. Use open slots for the highest-priority unblocked
  issues.
