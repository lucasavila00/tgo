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
conversion. Direct marshal benchmarks exclude the outer encoder. Direct unmarshal benchmarks
exclude the outer decoder.

## Allocation sources

The Go version, payload type, input data, and escape analysis determine the measured count.
Streaming methods avoid a complete envelope buffer or map when the wire form permits it.
Measured allocations can come from:

- public encoder or decoder state and interface conversion;
- marshaled result buffers;
- copied raw payload or content values and decoded member names;
- each untagged decode attempt;
- payload fields and custom methods;
- boxed payloads that escape and returned errors; and
- compatibility buffers and maps used by direct methods.

`budgets.json` is the only list of numeric limits. Keep it next to the runner because these values
measure this implementation. They are not part of the TGo language specification.
