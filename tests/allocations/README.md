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

## Allocation sources

The Go version, payload type, input data, and escape analysis determine the measured count.
The generated paths have these main allocation sources:

- External decoding uses a map, decoded keys, a copied `json.RawMessage`, and a payload decode.
- Internal encoding marshals the payload and joins the tag with the object. Internal decoding
  reads the tag, then decodes the original object.
- Adjacent decoding reads the tag, copies the content into a `json.RawMessage`, and decodes it.
- Untagged decoding tries payload decoders in declaration order. Each failed attempt can allocate.
- JSON names that cannot go in a struct tag use generated envelope buffers or compatibility maps.
- Payload fields, custom JSON methods, returned byte slices, decoder state, and errors can allocate.

`budgets.json` is the only list of numeric limits. Keep it next to the runner because these values
measure this implementation. They are not part of the TGo language specification.
