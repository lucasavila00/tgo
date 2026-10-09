# Tgolint fixture tests

The `CI / tests (tgolint-test)` hosted check runs all fixtures in this directory
for each pull request and each push to `main`. The workflow has no path filter,
so compiler, syntax, tgolint, fixture, and generated-output changes run this
check.

Run the same check locally with:

```sh
make tgolint-test
```
