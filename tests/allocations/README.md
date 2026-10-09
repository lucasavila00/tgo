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
The generated paths have these main allocation sources:

- Public `encoding/json` calls allocate encoder or decoder state and can convert the enum to an
  interface. Direct generated methods exclude the outer encoder or decoder.
- Byte-slice marshal methods allocate their returned buffer. Byte-slice external decoding uses a
  map and a copied raw payload. Adjacent decoding copies raw content. Internal decoding reads the
  tag and then decodes the original object.
- External and adjacent streaming writes emit the envelope around one payload encode.
- External streaming reads copy the selected payload. Adjacent streaming reads copy its content.
- Internal streaming writes can buffer the complete enum when the payload has a custom JSON or text
  method. Internal streaming reads buffer the complete value before normal decoding.
- Untagged decoding tries payload decoders in declaration order. Each failed attempt can allocate.
- Payload fields, custom methods, returned buffers, decoded names, copied values, boxes, and errors
  can allocate.

`budgets.json` is the only list of numeric limits. Keep it next to the runner because these values
measure this implementation. They are not part of the TGo language specification.
