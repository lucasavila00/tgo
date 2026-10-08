# Check Go callers

Run:

```sh
tgo build ./...
tgolint ./...
```

Fix every diagnostic. The command checks loaded Go packages for:

- invalid tgo zero values;
- constructor bypasses;
- unchecked `(T, error)` results;
- unchecked `(T, bool)` and comma-ok results;
- incomplete enum switches and wrong payload reads;
- changed, aliased, captured, or pointer enum receivers; and
- possibly nil or unknown pointers used as `%T`;
- unsafe `%T` zero values, literals, collections, calls, and function values; and
- the same errors through control flow, wrappers, embedding, and generics.

You can return an unchanged result pair. Otherwise, check `err` or `ok` before you
use the value. Do not take an address of a pending pair variable or capture it in a
closure. Keep both variables local to the function. Check the pair before a
`goto`, `break`, `continue`, or `fallthrough`.

Use a local value in an enum tag switch. Cover each numeric tag. Read only the
payload for that tag. Add a default that returns or panics.
Do not call generated `Tgo*` methods through a structural interface or an open
generic constraint.
Do not pass a tgo type to a generic function or method that can make its zero
value. The same rule applies when you save the function or method as a value.
`will` means the unsafe event is proved. `can` means a runtime value is unknown.
Local Boolean and integer assignments can change `can` to `will` or remove the
diagnostic. Package variables, captured values, addresses, narrowing conversions,
and unsupported expressions stay `can`. Fix both. Keep a generic function value local
so the linter can check each call.
Return a generic closure as a direct function literal. The checker does not yet
follow that closure through a local variable or another helper.

For `%T`, prove a possibly nil pointer non-nil before use. The checker follows nil comparisons,
Boolean guards, direct aliases, branches, loops, and early exits. A comma-ok map read or channel
receive proves a declared `%T` element only when `ok` is true. A pointer type assertion also needs
a nil check.

## Go boundary

TGo trusts values from Go. No generated validator checks the boundary. Go callers must use
constructors and must read enum payloads only after the matching tag check. `tgolint` reports
unsafe patterns that it can prove from Go source. It cannot inspect reflection, `unsafe`, cgo,
races, or foreign state.

`%T` has the Go `*T` representation. Unchecked Go can still pass nil. The linter adds no runtime
check and cannot prove code that runs through reflection, `unsafe`, cgo, or a data race.
