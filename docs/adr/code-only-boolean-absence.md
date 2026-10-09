# Use local contracts for Boolean absence

Status: Proposed

## Problem

A proposal adds postfix `?` for functions that return a final Boolean. Its examples are
[syntax queries](../../pkg/syntax/query.tgo#L135),
[effect fact decoding](../../internal/tgolint/generic_effects.tgo#L151), and
[nil path analysis](../../internal/tgolint/nil_facts.tgo#L216).

These Boolean results have different meanings. One operator would hide three API and control-flow
problems.

## Decision

Do not add Boolean propagation syntax for these examples. Fix each contract in code.

### Return one pointer from syntax queries

Every successful `*Of` query returns a non-nil pointer. Return `nil` for a node of another form.
The Boolean result repeats the pointer state.

```go
// Before
expression, ok := ExpressionOf(node)
if !ok { return nil, false }

// After
expression := ExpressionOf(node)
if expression == nil { return nil }
```

Callers use the same direct test:

```go
if node := syntax.DefaultExpressionOf(extension); node != nil {
	// Lower the default expression.
}
```

### Return an error from effect decoding

An unknown effect code is invalid wire data. It is not normal absence. Change the decoders to
return `error`, and use the existing error propagation operator.

```go
func decodeGenericEffects(encoded []genericEffectWire) ([]genericEffect, error) {
	effects := make([]genericEffect, 0, len(encoded))
	for _, item := range encoded {
		for _, condition := range item.Conditions {
			kind := decodeEffectConditionKind(condition.Kind)!
			// Append the decoded condition.
		}
	}
	return effects, nil
}
```

`decodeGenericEffectSet` propagates errors while it decodes all four lists. The fact import keeps
its current behavior with one check:

```go
decoded, err := decodeGenericEffectSet(fact)
if err != nil { return nil }
return decoded
```

Keep `effectKind` as the internal TGo enum. Keep its wire form as an integer because analysis facts
need exported data and generated enums keep their representation private.

### Build nil paths in one loop

`nilPlace` recurses only to remove parentheses, selectors, indexes, and dereferences. Use one loop.
Prepend each path part because the scan starts at the outer expression.

```go
// Before: each layer recurses and propagates failure.
base, ok := e.nilPlace(node.X)
if !ok { return nilPlace{}, false }
base.path += "/d"
return base, true

// After: one loop removes each layer.
path := ""
for {
	switch node := expression.(type) {
	case *ast.StarExpr:
		path = "/d" + path
		expression = node.X
	case *ast.Ident:
		return nilPlace{object: e.info.ObjectOf(node), path: path}, true
	// Parentheses, selectors, and indexes update expression and path here.
	}
}
```

The loop keeps the current checks. It removes only the repeated recursive result checks.

## Enum choice

An option enum does not improve these sites. A syntax pointer already has present and absent states.
An effect decoding failure is an error. A nil path has one success payload. The existing
`effectKind` enum remains useful because its valid meanings are a closed set.

## Migration and cost

Change the exported syntax query signatures and their callers in `pkg/syntax`, `internal/compiler`,
and `internal/tgolint`. Change the effect decoders to errors. Rewrite `nilPlace` and test path order
for parentheses, fields, indexes, and dereferences.

The syntax query change is an API break. Effect decoding allocates an error only for invalid data.
The nil path loop keeps one explicit failure return for each invalid form. No parser, compiler,
generated-code, or linter change is needed for a new operator.
