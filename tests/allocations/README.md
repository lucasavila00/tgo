# Allocation checks

`run.py` builds the current TGo compiler, generates a private copy of the business fixture,
and runs its JSON benchmarks. It compares the reported bytes and allocations per operation
with `budgets.json`.

Run the checks with:

```text
make allocation-test
```

The main CI target runs this check. A benchmark can use less memory than its budget. An increase
fails CI. Review the generated code and the benchmark result before you increase a budget.

Use 100,000 operations for each benchmark. This fixed count keeps the one-time benchmark setup
cost stable and keeps the check fast. The benchmark measures the public `encoding/json` call,
including the generated enum method.
