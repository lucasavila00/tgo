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

Treat a direct tgo parameter, interface assertion, callback result, decoder result, stored value,
or arbitrary `(T, error)` result as untrusted. Checking its original error is necessary, but it
does not validate the value. Call the generated validator and check that error before use:

```go
foreign, err := decode()
if err != nil {
    return err
}
value, err := model.ValidateEvent(foreign)
if err != nil {
    return err
}
```

The validator returns a reconstructed graph. It rejects invalid enum tags, mismatched payloads,
failed checked predicates, and nested invalid models. It copies arrays, slices, maps, pointers,
and supported interface values. It preserves repeated pointers, maps, and identical slice
headers, and it stops cycles. Overlapping slice views with different headers rebuild
independently. Later changes to the foreign graph do not change the rebuilt graph.

Nil values stay nil. A channel or function is shared only when its static type cannot transport
or return a tgo model. The validator rejects interface-bearing channel or function signatures,
unsafe pointers, and ordinary private fields that it cannot inspect. Check application nil
and ownership rules after validation. Shared ordinary data can still need an application copy.

`tgolint` recognizes generated validators across packages. It recognizes direct validator
wrappers and simple local function values. It treats the result as valid only after the matching
error is proved nil. It is local static analysis, not a runtime proof. It cannot inspect cgo or
`unsafe` memory, prevent races, or prove ownership of shared ordinary data.
