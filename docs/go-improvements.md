# Go improvements

## 1. Short error propagation

Add `value := call()?`, like [Rust](https://doc.rust-lang.org/std/result/#the-question-mark-operator),
or `value := try call()`, like [Zig](https://ziglang.org/documentation/master/#try).
Return errors without repeated `if err != nil` blocks. Keep existing error checks
for recovery and added context.

## 2. Types with fixed variants

Add types with a fixed set of variants. Each variant has its own fields.
A business account requires a company name; a personal account does not.
Require switches on these types to cover every variant. New variants expose
missing cases at compile time.

## 3. Checked value types

Add rules to value types: a quantity must be positive; a name must not be empty.
Check constants at compile time and external input at runtime. Prevent writes
that break the rule. Review one rule instead of repeated checks.

## 4. Typed DSLs

Add support for small DSLs that use Go types. A form could use an account type
to define its fields and input checks. Check DSL field names and types at compile
time. Make generated code available for review.

## 5. Collection queries

Add short expressions to filter, group, and sum collections. “Total sales per
customer” becomes one expression instead of a loop with map updates. Define
result types and overflow behavior. Keep database access explicit.
