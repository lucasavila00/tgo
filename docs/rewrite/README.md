# Rewrite: a language that builds Go packages

Use Go structs, functions, packages, and reference types. Add explicit initialization
and value enums. Keep direct Go calls both ways. Change syntax only where these rules need it.
Check source rules at compile time. Trust Go calls, as TypeScript trusts JavaScript calls.

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
Match uses the active payload; reference fields keep their aliases. Constructors zero inactive fields.
The compiler relies on this layout for equality. Foreign Go code can break that assumption.
No memory overlay: Go must see pointer fields. Storage is the sum of the payload fields.
This layout must meet the cost requirement below; it is not a claim of Rust-sized enums.
Go decides stack or heap placement. Recursive payloads need pointers, slices, or maps.
Use Go equality. With inactive fields zero, valid enums compare by tag and active payload.
All payloads must be comparable. Map or slice fields prevent equality and use as a map key.

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
Keep `append`, `copy`, indexing, and assignment. Type-check supplied values; do not scan them.

For element types with invalid zeros, reject `clear(slice)`. Allow reslicing only when the
compiler can prove the upper bound is within the current length. Otherwise reject the code.
This includes structs and arrays containing enums. Add no runtime length guard.
Other slices keep Go bounds. Grow enum slices with `append`.
Keep `delete` and `clear(map)`: they remove values.
Keep nil maps and slices distinct from empty ones. A write to a nil map still panics.

```text
if account, ok := accounts[id]; ok {
    use(account) // Bound only in the successful branch.
}
bad := accounts[id] // Error: a missing key would produce a zero enum.
```

Require this presence test for map reads, channel receives, and type assertions
when the result type has no valid zero. The test is ordinary Go; it adds no enum validation.
Zero-valid types keep normal Go reads.

## 4. Direct Go FFI, both ways

Import actual Go types and emit normal calls. Keep aliases, mutations, pointer identity,
cycles, nil values, retained callbacks, and interfaces. Do not silently copy collections.
Keep result order, partial results, error identity, typed nil, and ordinary panics.

```text
import "slices"
items := []int{3, 1}
slices.Sort(items) // Go mutates the same backing array.
```

Export model type names so Go can name parameters, containers, and callbacks.
Keep payload fields private. Constructors and accessors are ordinary Go functions.
Calls and reads add no validation, conversion scans, or copies.

```go
// Generated read and call: ordinary Go.
a := items[i]
name := Name(a)

// Generated match: dispatch once on the tag.
func Name(a Account) string {
    switch a.tag {
    case 1: return a.personal.Name
    case 2: return a.business.Company
    default: panic("invalid Account")
    }
}
```

The default arm handles an unmatched tag. It is not a separate validation pass.
Other foreign mistakes may produce wrong results without a panic. No detection guarantee.

```go
// Ordinary Go caller:
a := accounts.NewPersonal("Ana")
name := accounts.Name(a)
accounts.Name(accounts.Account{}) // Caller broke the contract; match panics here.
```

Go can also clear a shared enum slice or change a retained pointer. No later read check runs.
Private fields reduce accidents; they do not prevent zero values. Caller code must honor the model.
The same trust rule applies to interfaces, callbacks, channel values, and imported generic code.

```text
a, err := legacy.LoadAccount()
if err != nil { return err }
use(a) // Trust the Go result. No special slot or runtime validation.
```

## 5. Cost and safety contract

No added runtime cost compared with the equivalent handwritten Go design.
No automatic boundary checks, element scans, deep copies, or validation wrappers.
Keep ordinary Go bounds checks, nil behavior, interface operations, and written branches.
Only operations requested by the source should remain at runtime.

Compile-time rules catch mistakes in source code. They do not protect against Go callers.
This is the TypeScript/JavaScript tradeoff: foreign code can violate declared types and invariants.
The developer can write validation where needed. It is explicit code with an explicit cost.
Keep Go synchronization and memory rules. Do not treat a broken model as license for unsafe memory use.

Compare time, allocations, memory layout, and generated calls against handwritten Go.
Reject layouts or helpers that add cost. Sum-of-payload storage and accessor calls need this proof;
removing validation alone does not establish equal performance.

[Detailed rules, cases, and sources](../research/go/rewrite/notes.md).
