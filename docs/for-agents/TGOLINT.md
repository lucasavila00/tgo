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
closure.

Use a local value in an enum tag switch. Cover each numeric tag. Read only the
payload for that tag. Add a default that returns or panics.

## Foreign values

The linter checks that code handles errors from FFI calls, decoders, and callbacks.
It cannot prove that foreign code returned a valid value when the error is nil.

At ingress, check the value rule, enum tag and payload, nested tgo values, nil rules,
and ownership of mutable data. Build a new tgo value with generated constructors.
Copy mutable data when foreign code can change it later.
