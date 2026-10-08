# ADR: Use contextual error propagation expressions

Status: proposed

## Context

tgo keeps Go function signatures and calls Go code directly. A checked constructor and many Go
APIs return `(T, error)`. Callers repeat the same branch to add operation context.

Two designs can reduce this code. An expression operator can return the error from the current
function. A generic effect system can infer which functions fail and can change how calls work.
Effect inference would hide control flow and would need new rules for Go functions, callbacks,
interfaces, and generated constructors.

## Decision

Add a postfix propagation expression with required context. Do not add generic effect tracking.

```text
account := store.Load(id)? "load account"
```

The grammar is `PropagateExpr = CallExpr "?" string_lit .`. The context must be a nonempty
interpreted or raw string literal. A bare `?` is an error. The call and the enclosing function
must have the predeclared `error` type as their final result.

The first version permits one value before the error. It also permits an error-only expression
statement, such as `store.Flush()? "flush store"`.

The operator tests `err != nil`. On error, it returns `fmt.Errorf("%s: %w", context, err)`.
This adds the context once and keeps the cause for `errors.Is`, `errors.As`, and `Unwrap`.
It uses the normal Go typed-nil rule and does not inspect the concrete error.

On failure, each earlier result of the enclosing function is its Go zero. This rule also
applies to named results and ignores their current values. An invalid-zero tgo result can occur
only beside a non-nil error, as it can in a generated checked constructor. Callers must not use
that result.

## Generated Go

`account := store.Load(id)? "load account"` lowers to this shape:

```go
func LoadLabel(store Store, id ID) (string, error) {
	value, err := store.Load(id)
	if err != nil {
		var zero string
		return zero, fmt.Errorf("%s: %w", "load account", err)
	}
	account := value
	return Label(account), nil
}
```

The compiler selects fresh temporary names that do not occur in the source. It emits a local
zero variable for each earlier function result. Thus it also supports type parameters and
types whose composite literal is not available.

Lowering must preserve each evaluation that Go orders and must not evaluate a later operand
after an error. It must not use an immediately invoked function, `defer`, `panic`, or a helper
type. The output has the same work as a handwritten Go check with `fmt.Errorf` and `%w`.

## Expression limits

The first stage accepts `?` only when its call is the full right side of a single-value
assignment, a short declaration, or an expression statement. This rule keeps the lowering and
source positions clear.

A later stage can accept eager nested expressions, such as `Use(Load()? "load")`. The lowerer
must save earlier ordered operands and evaluate later operands only after success. Until that
lowerer exists, these positions are compile errors.

The operator is not allowed in the right operand of `&&` or `||`, in a `go` statement, in a
`defer` statement, or in a function literal that returns a different final result. It is also
not allowed at package scope. These limits prevent a rewrite from changing when code runs or
which function returns.

## Safety and compatibility

The check is visible in generated Go, so `tgolint` can follow the success and failure paths.
The source checker must type-check the call results and the enclosing function before lowering.
Diagnostics point to the `?` token.

The feature does not change a signature or infer an error. Go callers see the same function and
error result. Calls without propagation keep Go behavior. Existing files remain valid because
`?` was not valid Go syntax. The compiler adds `fmt` with a fresh import name.

The generated failure branch uses zero values only as return values paired with a non-nil
error. It does not store them in user variables. This keeps the current tgo rules for enum and
checked zero values.

## Why not generic effect tracking

An inferred effect would become part of function type identity. It would need rules for method
sets, interfaces, generic constraints, function values, and Go imports. It could also change a
public Go signature after a body edit. These changes conflict with direct Go compatibility.

The contextual operator shows the return point, adds a stable operation name, and keeps the
error chain and source signature. It maps to error handling that Go tools understand.

## Drawbacks

`?` is new syntax that Go editors do not parse before generation. A fixed string cannot include
a runtime identifier. A caller must add that detail before propagation when it is necessary.
The first stage does not handle complex expressions or calls with multiple value results.
Zeroing earlier named results can differ from a handwritten bare return.

## Staged decision

1. Add full-right-side and error-only forms with required context and `%w` wrapping.
2. Verify zero results, named results, typed nils, cause matching, and source positions.
3. Add eager nested positions only with evaluation-order and source-position tests.
4. Consider multiple success values only for matching multi-value assignments.
5. Keep short-circuit, `go`, and `defer` positions rejected unless exact lowering is proved.

Do not start an effect system as part of these stages. Revisit that choice only if a use case
cannot keep an explicit Go error result and cannot use this operator.
