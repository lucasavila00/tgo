# Mark an elided success error with a trailing comma

## Problem

Postfix `!` and `!!` remove repeated failure branches. Successful returns still repeat `nil`:

```go
func LoadName(repo Repo, id ID) (string, error) {
	user := repo.Find(id)!
	return user.Name, nil
}
```

Changing `return user.Name` would assign new behavior to Go syntax. It would also need type
information to distinguish one value from a call that returns the complete result tuple. TGo must
preserve valid Go source when it contains no TGo syntax.

## Proposal

A trailing comma in a non-empty return list marks one elided `nil`:

```go
func LoadName(repo Repo, id ID) (string, error) {
	user := repo.Find(id)!
	return user.Name,
}
```

The compiler lowers it mechanically:

```go
return user.Name, nil
```

The grammar extension is:

```text
TGoReturn = "return" ExpressionList "," .
```

The comma is required. Bare `return`, `return value`, and `return value, nil` keep their Go
behavior. All valid Go return statements remain unchanged.

The rule does not inspect the enclosing result list or the expression types. It works in a
function, method, or function literal when the lowered return is valid Go. The normal package
check reports a result count, assignment, or `nil` error at the original return when it is not
valid.

The compiler inserts exactly one `nil`. For example, `return first, second,` becomes
`return first, second, nil`. A call remains one expression, so `return Pair(),` becomes
`return Pair(), nil`; Go accepts or rejects that output by its normal return rules. A caller can
bind multiple results first when needed.

Postfix propagation lowers before the return is emitted:

```go
return repo.Find(id)!,
return repo.Find(id)!!,
```

Each propagation operator keeps its current failure behavior. The trailing comma adds only the
success `nil`.

The parser records the trailing comma as TGo syntax and creates the synthetic `nil` at the comma
position. No new type query, inference pass, helper, closure, or runtime operation is required.
Because the source contains an explicit TGo token, ordinary-Go byte identity does not apply to
this return.
