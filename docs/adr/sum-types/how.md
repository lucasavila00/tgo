# Sum types: HOW

## Approach

Generate private typed slots. For each identical field type, reserve its maximum
occurrence count within one variant. Share those slots across variants. Preserve
named-type and type-parameter identity before instantiation. Indirect recursion
uses Go references.

Assign tags in declaration order, starting at zero. For N variants, use the
smallest `uint8/16/32/64` that holds N−1. Order slots by first use; Go supplies
alignment. Constructors build zeroed structs, set the tag, and fill active slots.
Replacement copies whole structs, clearing inactive references. Go accessors
return `(payload, ok)`, zero/false on mismatch. Go callers must preserve valid tags.

## Example

```go
type Account struct { tag uint8; text string }
var label string
snapshot := account
switch snapshot.tag {
case 0: label = snapshot.text // Personal
case 1: label = snapshot.text // Business
}
```

## Tradeoffs

On amd64: 24 bytes versus 40 with separate string fields.
`uint64|float64` uses 24 bytes; safe scalar bit packing needs 16.
Typed slots favor direct operations over minimum size.

[Go scans fixed pointer maps](https://go.dev/src/runtime/mbitmap.go).
Byte unions hide references; general pointer/scalar reconstruction needs unsafe
operations. Interfaces can box;
[escape analysis](https://go.dev/doc/gc-guide#Escape_analysis) decides stack placement.
