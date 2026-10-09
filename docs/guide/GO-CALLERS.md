# Check Go callers

Run:

```sh
tgo build ./...
tgolint ./...
```

Fix every diagnostic. The command checks loaded Go packages for:

- invalid checked-struct zero values;
- constructor bypasses;
- writes or address-taking through checked-struct fields;
- unchecked `(T, error)` results;
- unchecked `(T, bool)` and comma-ok results;
- incomplete enum switches and payload reads without a variant proof;
- possibly nil or unknown pointers used as `%T`;
- unsafe `%T` zero values, literals, collections, calls, and function values; and
- sequential `iota` sets in handwritten `.tgo` files; and
- manual error returns that postfix `!` or `!!` replaces exactly; and
- the same errors through control flow, wrappers, embedding, and generics.

Construct an enum with its generated package function. The function name is
`New<Enum><Variant>`, and its arguments follow field declaration order:

```go
account := model.NewAccountPersonal("Lucas")
```

Do not construct a generated payload type and convert it to the enum. TGo source uses only
`Account.Personal{Name: "Lucas"}`.

You can return an unchanged result pair. Otherwise, check `err` or `ok` before you
use the value. Do not take an address of a pending pair variable or capture it in a
closure. Keep both variables local to the function. Check the pair before a
`goto`, `break`, `continue`, or `fallthrough`.

Use `switch value.Tag()` for an enum value or pointer. Use `exhaustive:` to require every declared
tag, or use `default:` for fallback behavior. Read a payload only when the clause flow has one
possible tag. A default has the union of omitted variants. A simple early-exit guard also proves a
variant: `if value.Tag() != EventTagStarted { return }`.
Do not call generated enum methods through a structural interface or an open
generic constraint.
Every payload call and payload method value needs one of these proofs. The receiver root must belong
to the current function. An assignment or writable escape ends the proof. Control flow must not
bypass the proof.
Do not pass a TGo type to a generic function or method that can make its zero
value. The same rule applies when you save the function or method as a value.
`will` means the unsafe event is proved. `can` means a runtime value is unknown.
Local Boolean and integer assignments can change `can` to `will` or remove the
diagnostic. Package variables, captured values, addresses, narrowing conversions,
and unsupported expressions stay `can`. Fix both. Keep a generic function value local
so the linter can check each call.
Return a generic closure directly, through a stable local alias, or through one
statically resolved forwarding helper. An unresolved return call that receives an
effectful closure keeps a conditional effect.

Replace a reported `iota` set with a TGo enum. Keep explicit integer conversion code when the
old values are part of a stored format, protocol, or Go boundary. Bit sets remain valid.

Replace a reported manual error branch with the postfix form in the diagnostic. The diagnostic
proves that the call, zero return values, error value or wrapper text, and control flow match the
generated code. Keep a manual branch when `tgolint` does not report it.

For `%T`, prove a possibly nil pointer non-nil before use. The checker follows nil comparisons,
Boolean guards, direct aliases, branches, loops, and early exits. A comma-ok map read or channel
receive proves a declared `%T` element only when `ok` is true. A pointer type assertion also needs
a nil check.

## Go boundary

TGo trusts values from Go. No generated validator checks the boundary. Go callers must use
constructors. `tgolint` requires a variant proof for each payload read. It cannot inspect
reflection, `unsafe`, cgo, races, or foreign state.

For a checked struct `Port`, call the generated `NewPort` function and check its error:

```go
port, err := model.NewPort(3)
if err != nil {
	return err
}
```

The constructor parameters follow field declaration order. `NewPort` and the generated staging
carrier are Go ABI. TGo source uses `model.Port{number: 3}` and cannot refer to either generated
name.

`%T` has the Go `*T` representation. Unchecked Go can still pass nil. The linter adds no runtime
check and cannot prove code that runs through reflection, `unsafe`, cgo, or a data race.

The linter has a fixed rule set. It supports `//tgolint:ignore` for one line and
`//tgolint:ignore-file` for one file. Do not use these directives in this repository's production
code. Fix the diagnostic.
