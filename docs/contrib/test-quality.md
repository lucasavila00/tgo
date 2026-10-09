# Test quality

Before you add a regression test, search the repository for the same behavior and
failure mode. Extend an existing table or fixture when it runs the same path and
would fail for the same defect.

Keep a separate test when it proves a distinct contract. Examples include a
different package or integration layer, source form, diagnostic position, build
tag, platform, generated output, allocation limit, or protocol behavior. A focused
unit test and an end-to-end test can remain separate when they test different
contracts.

Do a periodic test review. Group tests by the contract that they prove, then remove
or combine only cases that have stronger retained coverage. For each removal,
record the retained test in the pull request. Report test counts and hosted CI job
durations before and after the change. Do not use a test-count limit.
