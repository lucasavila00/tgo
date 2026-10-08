# Rewrite: a language that builds Go packages

Use Go structs, functions, packages, and reference types. Add explicit initialization
and value enums. Keep direct Go calls both ways. Change syntax only where these rules need it.

## 1. Explicit initialization

Require an initializer for each variable and every field in a struct literal.
`..default` fills omitted fields with declared defaults. Defaults run per construction,
not once per type. Literal maps and slices allocate fresh storage.

```text
type Request struct {
    ID string
    Tags []string = []string{}
    Counts map[string]int = map[string]int{}
}
a := Request{ID: "a", ..default}
b := Request{ID: "b", ..default}
a.Counts["x"] = 1 // b.Counts stays empty.
full := Request{ID: "c", Tags: nil, Counts: nil} // Explicit nil is valid.
bad := Request{ID: "d"} // Error: omitted fields need ..default.
missing := Request{..default} // Error: ID has no default.
```

Evaluate explicit fields in source order, then selected defaults in declaration order.
Defaults can use normal expressions but cannot read the object under construction.
A default that names shared storage still shares it. A copied struct also shares map
and slice data. Keep Go field assignment. Initialization does not mean immutability.

## 2. Value enums with collection payloads

Allow structs, arrays, slices, maps, pointers, interfaces, channels, and functions in payloads.
Require every match case. Enums have no valid zero or nil value.

```text
type Account enum {
    Personal struct { Name string }
    Business struct { Company string; Members []Account; Tags map[string]string }
}
a := Account.Business{Company: "Acme", Members: []Account{}, Tags: map[string]string{}}
match a {
case Personal(p): use(p.Name)
case Business(b): use(b.Members)
}
```

Emit a tagged Go struct with private fields. Each variant gets separate typed payload fields.
Tag zero is invalid. Inactive fields contain Go zeros but are not source values.

```go
type Account struct {
    tag uint8
    personal struct { Name string }
    business struct {
        Company string
        Members []Account
        Tags map[string]string
    }
}
```

Constructors set the tag and active payload. Replacing a variant replaces the whole value.
Match copies the active payload; reference fields keep their aliases. Check active field values.
Require inactive storage to stay zero. This also keeps Go map-key equality consistent.
No memory overlay: Go must see pointer fields. Storage can exceed Rust enum size.
Go decides stack or heap placement. Recursive payloads need pointers, slices, or maps.
Compare the tag and active payload only. All payloads must support comparison.
Map or slice fields make the enum non-comparable; it cannot be a map key.

## 3. Slices and maps without invalid zero elements

A zero is valid for scalars and nil-able Go types. It is invalid for enums.
A struct or array has a valid zero only if each contained value does.
Declared field defaults are separate from this rule.

```text
items := make([]Account, 0, 8) // Capacity only; no visible elements.
items = append(items, Account.Personal{Name: "Ana"})
bad := make([]Account, 8) // Error: zero enum elements.
counts := make([]int, 8) // Explicit zero fill; valid.
accounts := make(map[string]Account) // Empty; no values to initialize.
```

`make([]T, n)` and `new(T)` explicitly request zeros, not field defaults.
Permit zero fill only when the produced values are valid. This includes structs
whose fields all accept zero. Otherwise, slice length must be the constant 0.
Array and slice literals must supply every element; no holes.
Keep `append`, `copy`, indexing, and assignment. Supplied values must be valid.

For element types with invalid zeros, reject `clear(slice)` and reslicing past `len`.
This includes structs and arrays containing enums. Even restoring a prior length is disallowed.
Other slices keep Go bounds. Grow enum slices with `append`.
Keep `delete` and `clear(map)`: they remove values.
Keep nil maps and slices distinct from empty ones. A write to a nil map still panics.

```text
if account, ok := accounts[id]; ok {
    use(account) // Bound only in the successful branch.
}
bad := accounts[id] // Error: a missing key would produce a zero enum.
```

Use this checked binding for map misses, closed-channel receives, and failed type assertions
when the result type has no valid zero. Zero-valid types keep normal Go reads.

## 4. Direct Go FFI, both ways

Import actual Go types and emit normal calls. Keep aliases, mutations, pointer identity,
cycles, nil values, retained callbacks, and interfaces. Do not silently copy collections.
Keep result order, partial results, error identity, typed nil, and ordinary panics.

```text
import "slices"
items := []int{3, 1}
slices.Sort(items) // Go mutates the same backing array.
```

Go can insert invalid enums through shared storage. Checking a container once is not enough.
Check each enum on every source read, including a local that Go can change through a pointer.
Copy the value, then check its tag and inline active fields. Check referenced values when read.
Bulk copies check the copied elements and keep Go overlap behavior.

```go
// Generated read from a shared slice:
a := checkAccount(items[i])
// Generated exported function:
func Name(a Account) string {
    return name(checkAccount(a))
}
```

`checkAccount` returns a valid value or panics with `ffi.InvalidValue`.
It does not change the Go signature or replace the function's own error result.
Export model type names so Go can use them in signatures, containers, and callbacks.
Keep representation fields private. Provide constructors and checked variant accessors:

```go
a := accounts.NewPersonal("Ana")
name := accounts.Name(a)
accounts.Name(zeroOf(a)) // ffi.InvalidValue, before source code uses the value.
```

Go can create `Account{}` or use `zeroOf[T any]`. Checked reads enforce validity, not name privacy.
Go can also clear an aliased enum slice after a call. The next source read must check again.
A bad later element can fail after earlier effects. Calls are not atomic or transactional.

Keep imported results in foreign slots until read. Checking an error does not read an enum slot.

```text
a, err := legacy.LoadAccount()
if err != nil { return err } // No read of a.
use(a) // Check a here. A valid partial value is also usable when err != nil.
```

Reading an invalid enum always fails. Error status does not prove value validity.

## 5. Safety boundary

The guarantee: source code never uses an invalid enum as a valid variant.
Shared Go storage can hold invalid values. Detect them on read; do not claim they cannot exist.
Keep Go synchronization duties. Checks do not fix races, unsafe writes, or arbitrary Go bugs.
Opaque Go objects retain their own rules. Initialization history cannot be recovered from Go values.

Require initializers in source code. At FFI, check value rules, not whether Go wrote each field.
Interface assertions into model types need checks. Callback adapters apply the same rules.
Unchecked FFI explicitly drops these guarantees. Representation fields stay private.

[Detailed rules, cases, and sources](../research/go/rewrite/notes.md).
