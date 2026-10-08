# ADR: Add `%T` non-null pointers

Status: proposed.

## Decision

Add `%T` as a non-null pointer to `T`. Keep `*T` as a nullable Go pointer.

```text
type User struct {
    Manager *User
    Account %Account
}

func LoadAccount(id ID) (%Account, error)
```

`Manager` can be nil. `Account` must be non-nil. Both use the Go `*T` representation and ABI.

The grammar addition is:

```text
NonNullPointerType = "%" Type .
```

Pointer markers compose:

```text
*T   nullable pointer to T
%T   non-null pointer to T
*%T  nullable pointer to a non-null pointer to T
%*T  non-null pointer to a nullable pointer to T
%%T  non-null pointer to a non-null pointer to T
```

## Static analysis only

`%T` generates no wrapper, constructor, runtime check, panic, or support function. It only adds
a static contract. The TGo compiler parses `%T` and erases it to `*T` in the normal Go output.
The compiler inserts no checks and performs no null-flow analysis.

`tgolint` owns all `%T` control-flow analysis. It analyzes both `.tgo` and `.go` source and
reports each error at its source position. This design has one flow engine. The compiler does
not embed or invoke that engine.

`tgolint` package facts record `%T` in fields, parameters, results, aliases, nested types, and
function types. These facts keep the contract across package boundaries. They are analysis
metadata, not generated support code.

## Type rules

The zero value of `%T` is invalid. `&value`, `new(T)`, and a result declared as `%T` are
non-null. A `%T` value can flow to `*T`. A `*T` value can flow to `%T` only when the current
control-flow path proves that the value is non-null.

The checks apply to declarations, assignments, returns, call arguments, receivers, field
writes, composite literals, collection writes, closures, and function values. Nested pointer
types keep the contract for each pointer level.

## Control-flow analysis

The checker narrows pointer types on each control-flow path, in the same way that TypeScript
narrows union types. It tracks `nil`, non-null, and unknown states for stable expressions.

```text
account := legacy.LoadAccount(id)
if account == nil {
    return ErrMissingAccount
}
use(account) // account is non-null here
```

The analysis includes these proofs:

- `value != nil` proves non-null on the true path.
- `value == nil` proves non-null on the false path.
- `!`, parentheses, `&&`, and `||` preserve facts on their exact short-circuit paths.
- Boolean guards such as `valid := value != nil` preserve the linked fact while stable.
- An early `return` or `panic` removes its path from later joins.
- A `break`, `continue`, or `goto` carries the current facts to its control-flow target.
- A nil `switch` case narrows the value in that case and in the remaining cases.
- A join keeps a fact only when every incoming path has that fact.
- A loop computes facts to a fixed point across its entry, back edges, and exits.

Direct local aliases share facts while their relationship is stable:

```text
candidate := account
if candidate != nil {
    use(account) // account and candidate are non-null here
}
```

An assignment ends facts about the old value and breaks its alias links. It does not end a
fact for an unchanged local copy. Passing a pointer value to a call does not end its local
fact. Taking the address of the pointer variable or capturing it in a closure ends the fact
when that code can assign to the variable.

The checker can narrow a field or indexed value while its storage is stable. An assignment to
the storage, an alias that can write it, or a call that can change it ends the fact. The checker
reports an error when it cannot prove that a value is non-null at a `%T` use.

## Maps, assertions, and channels

A one-result map read can return the zero value for a missing key. It is nullable unless the
analysis already proves that the key is present. A comma-ok read links the value to `ok`.

```text
account, ok := accounts[id]
if !ok {
    return ErrMissingAccount
}
use(account)
```

For `map[K]%T`, `ok` proves that the entry exists, and the map contract proves that its value
is non-null. For `map[K]*T`, the code must also prove `account != nil`. The checker follows both
facts through Boolean aliases, short-circuit conditions, early exits, and joins.

A known key stays present only while the key and map stay stable. An insertion can establish
the fact. A delete, clear, reassignment, or call that can change the map ends it.

A successful pointer type assertion proves the dynamic type, but it does not prove non-null.
An interface can contain a typed nil pointer. Code must prove both `ok` and `value != nil`
before it uses the asserted pointer as `%T`.

A channel receive follows the same rule as a map read. For `chan %T`, a true `ok` result proves
that a non-null value arrived. For `chan *T`, the code must also prove that the value is non-null.

## Go boundaries

Go source uses `*T`. `tgolint` reads the `%T` facts and applies the same flow rules to Go calls,
returns, fields, collections, and function values.

Unchecked Go, reflection, `unsafe`, cgo, and data races can break the contract. The feature adds
no runtime defense against these operations. Checked code must prove a foreign pointer non-null
before it flows to a `%T` position.
