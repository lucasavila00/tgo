# Autonomous Agent Coordination

Use these instructions when you coordinate autonomous work across multiple
agents. Move the requested work to review without unnecessary waiting.

- Define the complete acceptance criteria before implementation. Keep the
  active pull request within that scope. Create a separate issue for an
  unrelated finding.
- The root agent coordinates the work, defines the scope, tracks hosted CI,
  and reviews repository state. It must not create or edit repository files.
- Any subagent that creates or edits repository files must use the
  `gpt-5.6-sol` model. Give each implementation subagent the full task context.
- The runtime has four total agent slots. The root agent uses one slot, so
  three subagent slots are available.
- Use slots in short cycles. One subagent implements one issue, runs focused
  checks, pushes a complete draft pull request, reports its exact head commit,
  and then stops.
- The root agent tracks hosted CI and review without keeping the implementation
  subagent active. Immediately use the free slot for the highest-priority
  unblocked issue. Do not wait for the prior pull request to merge.
- Use a separate agent for the final independent review. Reviewer agents can
  use any available model. The main coordinating agent selects the reviewer
  model based on task risk, scope, and speed. The reviewer must remain
  independent from the implementer and must not edit the reviewed branch. The
  reviewer reports the result and then stops.
- Use hosted CI for full validation. Do not use an agent slot to wait for
  hosted CI.
- Obtain the exclusive local validation lease before you run a direct Go or
  lint command. Do not run another compile at the same time. Limit Go command
  parallelism to the capacity that the coordinator assigns. The locked
  Makefile build, test, lint, and code-generation targets obtain this lease
  automatically across all worktrees. On a local host, the common runner also
  needs 6 GiB of available memory before it starts. It limits its process tree
  to 4 GiB and disables swap use for that tree. This leaves at least 2 GiB
  available at admission time. Processes outside the runner can still use the
  reserve. Use `TGO_LOCAL_VALIDATION_MIN_AVAILABLE_KB` only when the local host
  needs a different admission threshold. The runner stops if the host cannot
  enforce the process-tree limit.
- Do not merge `main` into a clean pull request only to refresh its ancestry.
  This repository squash-merges pull requests. Merge `main` only when GitHub
  reports a conflict or Lucas asks. Preserve both sides of a conflict. Do not
  rebase unless Lucas explicitly asks to rewrite history.
- Do not use a GitLab-style merge train that repeatedly merges the latest
  `main` into every open pull request. A change to `main` does not require a
  branch update, a new review, or another CI run for a clean pull request.
  Do not require CI tied to an exact head commit as a separate readiness gate.
  Run the full tests again on the combined `main` branch before publishing.
- Keep a pull request in draft while implementation, conflict resolution,
  review, or required CI is incomplete.
- Mark a pull request ready as soon as its requested scope is complete, its
  independent review has no blocker, GitHub reports no
  conflict, and all required hosted checks pass.
- Rebuild the work inventory after a merge or a new issue changes priorities.
  Do not repeat the inventory while the external state is unchanged.
- Keep no more than six pull requests open at one time. Draft pull requests can
  outnumber active agents. Use open slots for the highest-priority unblocked
  issues.

## Correct the Model Before Repeating Fixes

The coordinator must detect repeated failures without waiting for Lucas to
intervene. If review finds a second missed case from the same analysis model,
or fixes require more special cases without a clear rule, pause implementation
and keep the pull request in draft. Preserve the work and free the agent slot.

Ask an independent architecture agent to inspect the code and failure evidence.
Choose a model suited to this work, such as `gpt-6-astra` or `gpt-6.1-sol`.
The agent must define the identities, state, transfer rules, control-flow joins,
and package summaries needed for the feature. It must assess existing Go and
Go tools code for reuse and explain any required TGo-specific logic.

The target is complete support for the intended language feature. Do not change
the language, reject valid programs, add a fallback, suppress diagnostics, or
weaken tests to make the implementation pass. Require the model to catch unsafe
programs and accept safe programs under the specification.

Before implementation resumes, check the proposed model against all known
failures and a table of relevant operations and boundaries. Include aliases,
mutation, branches, loops, calls, and package boundaries where they apply.
Require both safe and unsafe cases. Give a `gpt-5.6-sol` implementer the model,
the full acceptance criteria, and the agreed tests. The independent reviewer
must test the model across those cases, not only the last reported failure.

Keep architecture work focused on the feature. Do not add speculative systems
or redesign adjacent code. Resume implementation when the model explains the
required behavior, then verify the complete feature before marking it ready.
