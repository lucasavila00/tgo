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

Both pair variables must belong to the current function. This rule prevents a
failed call from publishing its zero result through a package variable or a
captured variable. A `goto`, `break`, `continue`, or `fallthrough` can move the
pair only after the proof.

The checker follows `if` conditions, Boolean `&&` and `||`, loops, switches, type
switches, fallthrough, and selects. It joins all paths that can continue. An
address or closure must not alias a pending value, error, or `ok` variable.

The checker rejects generated `Tgo*` access through a structural interface or
an open type parameter. These types erase the generated model identity. Exact
model constraints keep the normal exhaustive switch checks.

The checker records effects for generic functions and methods. An effect states
that a type argument can get a zero value or lose its model identity. The facts
cross package boundaries and pass through generic wrappers.

`will` means that the call arguments and path prove the unsafe event. `can` means
that a runtime value or path is unresolved. Both diagnostics reject the call.
The checker suppresses the diagnostic when the call proves that the event cannot
occur. It checks local function and method values at their call sites. It rejects
a value that escapes before its effects can be checked.

Returned-function effects are complete only for a function literal returned
directly. A closure returned through a local variable or another helper can hide
an effect from the checker.

An enum payload read needs an exhaustive switch on the same stable value. The
default must not continue. Internal loop breaks and internal `goto` targets are
valid. A branch that escapes the default is invalid.

## Runtime boundary

A nil error from an arbitrary FFI function, decoder, wrapper, or callback does not prove that
its tgo value is valid. `tgolint` keeps that value untrusted after the error check. Pass it to
the generated validator and check the new error:

```go
foreign, err := decode()
if err != nil {
    return err
}
value, err := model.ValidateEvent(foreign)
if err != nil {
    return err
}
use(value)
```

The same rule applies to an exact tgo model received as a Go parameter. A successful type
assertion proves the dynamic Go type. It does not prove the tgo tag, payload, or predicate.
Validate the asserted value before use.

The linter identifies exact generated validators with analysis facts. The facts cross package
boundaries. It also recognizes a direct wrapper that returns a generated validator or checked
constructor call, and a simple local function value that names such an operation. The value is
trusted only on a path where the matching error is proved nil.

The generated validator rejects invalid tags, rebuilds the active enum payload, reruns checked
predicates, validates nested models, and copies reachable pointers, slices, maps, and interfaces.
It preserves repeated pointers, maps, and identical slice headers, and it stops cycles. It
rejects ordinary private fields, unsafe pointers, channels, functions, or dynamic interface
values when their static shape can hide or transport a tgo model.

The linter does not inspect cgo or `unsafe` memory. It does not prove that shared ordinary data
has the required ownership, and it cannot prevent a race after validation. It follows the
supported local typed syntax and imported facts. A tool failure, an unanalyzed package, or an
unsupported wrapper shape gives no proof.
