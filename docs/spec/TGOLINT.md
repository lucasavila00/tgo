# Go safety checks

Run these commands after a tgo source change:

```sh
tgo build ./...
tgolint ./...
```

`tgolint` checks TGo source and ordinary Go code that uses generated tgo types.
It gets model data from generated packages and uses it in packages that import
them. It does not check generated files.

The compiler does not enforce these usage rules. A build can succeed when
`tgolint` reports a policy error.

## Checked by tgolint

A clean run means that the loaded Go packages do not contain these errors:

- a TGo variable declaration without an initializer;
- a TGo struct, array, or slice literal with an omitted field or element;
- a TGo named result read or returned before assignment;
- an invalid checked-struct zero from a declaration, named result, literal, `new`, `make`,
  `clear`, map read, channel read, type assertion, or longer reslice;
- a direct checked-struct literal in a Go file;
- a write or address-taking operation through a checked-struct field;
- a new defined Go type or conversion that bypasses tgo validation;
- direct access to private enum representation;
- a tgo result used before its matching error is proved nil;
- a presence result used before its matching `ok` value is proved true;
- a missing enum tag case, wrong payload read, unsafe default, or `fallthrough`;
- a wrong enum payload call in a recognized tag switch; or
- a `%T` value that is nil, unknown, zero-filled, omitted, or lost at a control-flow join;
- a map read, channel receive, or pointer assertion used without its required proof; or
- a sequential `iota` set that uses one defined integer type in handwritten TGo source; or
- a manual error return that has the exact behavior of postfix `!` or `!!`; or
- an explicit final `nil` return value that can use TGo's trailing comma; or
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

For `%T`, the checker also follows direct pointer aliases and Boolean guard aliases. A comma-ok
map read or channel receive proves a `%T` value only on the true path. A successful pointer type
assertion proves the type but not the value because the interface can contain a typed nil.

The checker exports `%T` paths for fields, parameters, results, aliases, nested collections, and
function types. It checks the same paths in importing Go packages. `%T` generates the same Go
pointer as `*T`; these facts exist only during analysis.

A structural interface or an open type parameter erases the generated model identity. The
checker does not recognize its tag switch. Exact model constraints keep the normal exhaustive
switch checks.

The checker records effects for generic functions and methods. An effect states
that a type argument can get a zero value or lose its model identity. The facts
cross package boundaries and pass through generic wrappers.

`will` means that the call arguments and path prove the unsafe event. `can` means
that a runtime value or path is unresolved. Both diagnostics reject the call.
The checker suppresses the diagnostic when the call proves that the event cannot
occur. It checks local function and method values at their call sites. It rejects
a value that escapes before its effects can be checked.

A forward control-flow analysis tracks local Boolean and integer assignments and
direct parameter aliases. It keeps a fact at a join only when every incoming path
agrees. It removes a branch edge when the condition is a proved constant. Taking
an address or creating a closure that captures the value removes the fact. Package
variables, free variables, narrowing conversions, and unsupported expressions stay
unresolved. Passing a Boolean or integer by value does not remove its fact.

An imported model fact includes its package path and type name. The package path
must match the object that owns the fact. Equal type names from different packages
remain different models.

Returned-function effects follow a direct function literal, stable local aliases,
and one statically resolved local helper. If an unresolved helper can return a
visible effectful closure, the checker records a conditional effect. It then rejects
an unsafe call or escape.

An enum payload read needs a tag switch on the same syntactic receiver. A TGo `exhaustive:` clause
requires all declared tags and emits the generated `UnknownTag` panic and required comment. A normal
default clause is fallback behavior and can cover omitted tags. A clause assignment to the receiver
or its selector prefix removes the clause proof. Each clause has the union of its possible variants.
A default has the union of omitted variants and proves a payload when only one variant remains.
A function literal does not inherit the proof. Direct `go` and `defer` calls do inherit it.
The checker does not analyze `Tag` calls, payload calls, or payload method values outside a
recognized canonical switch.

The `iota` modernization check requires two or more unique values. Their values must increase by
one from one common offset. Every value must come from `iota` or its repeated expression. The
check excludes bit shifts and bitwise expressions because they can define combinable flags. It
does not check `.go` files and does not offer a fix because integer values can cross a boundary.

The error-return modernization check reports two adjacent statements in handwritten `.tgo`
source. The first statement must declare only new variables from one static call. The second must
check its error and return the same zero values. Its final value must be the same error for `!!`,
or the `fmt.Errorf` wrapper that `!` emits. The function must have unnamed results that end in the
Go `error` type. The local error variable must have no use after the branch.

The wrapper text must contain the full static call name and `: %w`. Thus, `repo.Find(id)!` matches
`fmt.Errorf("repo.Find: %w", err)`. It does not match `fmt.Errorf("Find: %w", err)`. The check does
not report function values, assignments to existing variables, extra branch work, named results,
nonzero returns, or different error text. It does not check `.go` files or offer a fix.

The successful-return modernization check reports `return value, nil` in handwritten `.tgo`
source. It also accepts parenthesized `nil` and returns with more than two values. It does not
report one-result returns, a non-final `nil`, a shadowed `nil`, or an existing trailing comma.

The same check reports an explicit failure return when every result before the final error is the
exact zero for its declared type. For example, it reports `return nil, err`, `return 0, err`, and
longer zero prefixes in favor of one leading comma per zero result. It does not report existing
leading commas, a nonzero prefix, a final `nil`, or a value that it cannot prove is zero.

## Go boundary

The linter trusts exact TGo values that enter from Go parameters, calls, callbacks, decoders,
storage, and type assertions. It does not require runtime validation.

It still reports unsafe construction, invalid zero values, unchecked constructor result pairs,
incomplete tag switches, and wrong payload access when Go source proves the error. It cannot
inspect reflection, `unsafe`, cgo, races, or foreign state.

It also reports a possibly nil or unknown pointer at a `%T` use. It trusts a `%T` value returned by a
checked signature. Unchecked Go can still return nil, change storage after analysis, or create a
typed nil through reflection. These operations are outside the proof.

## Suppression

The rule set has no configuration. `//tgolint:ignore` suppresses diagnostics on its line and the
next line. `//tgolint:ignore-file` suppresses diagnostics in its file. Both directives work in Go
and TGo source.
