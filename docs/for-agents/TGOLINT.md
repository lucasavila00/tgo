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
Fix both. Keep a generic function value local so the linter can check each call.
Return a generic closure as a direct function literal. The checker does not yet
follow that closure through a local variable or another helper.

## Foreign values

The linter checks that code handles errors from FFI calls, decoders, and callbacks.
It cannot prove that foreign code returned a valid value when the error is nil.
It also cannot validate a tgo value imported by a successful type assertion.

At ingress, check the value rule, enum tag and payload, nested tgo values, nil rules,
and ownership of mutable data. Build a new tgo value with generated constructors.
Copy mutable data when foreign code can change it later.
