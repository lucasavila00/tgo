# WHAT

Require explicit initialization in TGo and prove that each operation uses
initialized, available values. Analyze all reachable code, including Go
libraries. Report a forbidden dependency result at its use in the selected TGo
package; do not require changes to the library or ban its whole package.

Every local and package variable declaration needs an initializer. Forbid
named results in TGo declarations, function literals, and function types.
Parameters, receivers, range bindings, and receive bindings get values from
their operations; those values must pass the same analysis.

An explicit `0`, `false`, empty string, or nullable `nil` is an initialized
value. An implicit default is not. Initialization does not override type
validity: an enum needs a variant, a checked value must pass its condition,
and a non-null value cannot be nil. Struct literals must supply every field;
array and slice literals must supply every element below their length.
Apply these rules through nested types, aliases, and generics.

Allocation can reserve storage. Prove initialization before a read, including
loop fills and writes through aliases. Check `new`, `make`, `clear`, `copy`,
`append`, reslicing, map reads, channel receives, assertions, and calls.
Empty collections remain valid. Accept every initialization pattern that the
analysis can prove; do not require one particular guard or constructor form.

Failed calls and unsuccessful presence operations leave their value results
unavailable. Prove success before use. Forwarding a result with its status
preserves this condition. Generated failure slots keep the Go ABI and do not
become valid source values.

Produce a deterministic inventory for every operation: possible zeros, type,
source position, cause, path condition, and generic condition. Include explicit
zeros and unknown values. Separate this inventory from errors: a supplied
scalar zero is valid, but default-filled or unavailable storage is not.

Analyze indirect calls, closures, loops, aliases, generic instantiations, and
reachable dependencies. Unknown effects cannot establish initialization.
`unsafe` and cgo can break the guarantee; record these boundaries.

Track implementation in [issue #266](https://github.com/lucasavila00/tgo/issues/266).
[HOW](how.md) defines the analysis; [PROOF](proof.md) defines acceptance.
