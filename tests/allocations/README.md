# Allocation checks

`run.py` builds the current TGo compiler, generates a private copy of the business fixture,
and runs its JSON benchmarks. It checks public `encoding/json` calls and direct generated
method calls. It compares the reported bytes and allocations per operation with `budgets.json`.

Run the checks with:

```text
make allocation-test
```

The main CI target runs this check. A benchmark can use less memory than its budget. An increase
fails CI. Review the generated code and the benchmark result before you increase a budget.

Use 100,000 operations for each benchmark. This fixed count keeps the one-time benchmark setup
cost stable and keeps the check fast. Public marshal benchmarks include the enum-to-interface
conversion. Direct method benchmarks measure the returned-buffer API without the outer encoder.
