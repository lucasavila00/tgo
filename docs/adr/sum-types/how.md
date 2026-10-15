# Sum types: HOW

## Approach

Generate private typed slots. For each identical field type, reserve its maximum
occurrence count within one variant. Share those slots across variants. Preserve
named-type and type-parameter identity before instantiation. Indirect recursion
uses Go references.

Use the smallest `uint8/16/32/64` tag that fits all variants plus zero for nil.
Put slots in first-use order; Go supplies alignment. Constructors build zeroed
structs and fill active slots. Replacement copies whole structs, clearing inactive
references. Go accessors return `(payload, ok)`, zero/false on mismatch.

## Example

```go
type Account struct { tag uint8; text string }
var label string
snapshot := account
switch snapshot.tag {
case 0: label = "none"
case 1, 2: label = snapshot.text
}
```

## Tradeoffs

On amd64, this uses 24 bytes versus 40 with separate string fields.
`uint64|float64` needs 24 bytes versus a 16-byte overlapping union.

[Go scans fixed pointer maps](https://go.dev/src/runtime/mbitmap.go).
Byte unions hide references; canonical pointer/scalar slots require unsafe
reconstruction. Interfaces can box payloads. Typed slots keep Go operations;
[escape analysis](https://go.dev/doc/gc-guide#Escape_analysis) decides stack placement.

Universal nil accepts Go zeros at one extra case. Nonzero enums need
initialization proofs and optional wrappers. An explicit empty default saves
cases but forces a payload-free domain choice.
