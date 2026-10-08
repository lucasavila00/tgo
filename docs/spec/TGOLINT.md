# Go safety checks

Run these commands after a tgo source change:

```sh
tgo build ./...
tgolint ./...
```

`tgolint` checks ordinary Go code that uses generated tgo types. It gets model
data from generated packages and uses it in packages that import them. It does
not check generated files.

## Checked by tgolint

A clean run means that the loaded Go packages do not contain these errors:

- an invalid tgo zero from a declaration, named result, literal, `new`, `make`,
  `clear`, map read, channel read, type assertion, or longer reslice;
- a new defined Go type or conversion that bypasses a tgo constructor;
- direct access to private generated representation;
- a tgo result used before its matching error is proved nil;
- a presence result used before its matching `ok` value is proved true;
- a missing enum tag case, wrong payload read, unsafe default, or `fallthrough`;
- an enum receiver that is a pointer, alias, capture, or changed value; or
- the same errors hidden by embedding, wrappers, function values, control flow,
  or generic constraints.

The result rule applies to each call that returns `(T, error)` when `T` is a tgo
model or contains a tgo model. The presence rule applies to `(T, bool)` calls,
map reads, channel reads, and type assertions. A result pair can be returned
without a local check. Otherwise, the code must prove `err == nil` or `ok == true`
before it uses the value.

The checker follows `if` conditions, Boolean `&&` and `||`, loops, switches, type
switches, fallthrough, and selects. It joins all paths that can continue. An
address or closure must not alias a pending value, error, or `ok` variable.

An enum payload read needs an exhaustive switch on the same stable value. The
default must not continue. Internal loop breaks and internal `goto` targets are
valid. A branch that escapes the default is invalid.

## Runtime boundary

The linter checks that Go code handles an error from an FFI function, decoder,
wrapper, or callback. It cannot prove that the function follows its contract. A
function can return an invalid tgo value with a nil error.

The linter also cannot inspect a value made by cgo, `unsafe`, reflection, storage,
or code outside the loaded packages. It cannot prevent a race or a later change
through shared mutable data.

A boundary module must check these facts at ingress:

- each checked value satisfies its predicate;
- each enum tag is known and matches its active payload;
- each nested tgo value is valid;
- each application nil rule holds; and
- foreign code cannot change mutable data after the check.

Decode into transport types. Construct checked values and enum variants with their
generated constructors. Check each error. Copy maps, slices, pointers, or interface
data when foreign code can change them. Apply the same checks before storage,
encoding, cgo calls, and callbacks.
