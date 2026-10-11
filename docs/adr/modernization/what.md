# WHAT

This defines planned tooling. Start with a Go package:

1. Rename every handwritten `.go` source and test file to
   `.tgo`. Keep build constraints, file suffixes, imports, and module paths.
2. Compile the package. Correct compiler errors.
3. Run `tgolint`. Apply modernization diagnostics, compile, and
   run tests. Repeat until diagnostics are clear.

Generated Go files are outputs. Other packages can
remain Go. Compiler errors concern language validity. Linter errors identify
equivalent uses of supported TGo features. Valid Go forms remain valid syntax.

For example:

```go
value, err := load()
if err != nil {
    return 0, err
}
```

becomes:

```text
value := load()!!
```

Require the same error and zero failure results. Use `!` only when the
existing wrapper matches its error text and wrapping behavior exactly.

For [if expressions](../value-expressions/what.md), report a local declaration
followed by branches that each assign one value to that local:

```text
var label string
if active { label = "yes" } else { label = "no" }
```

becomes `var label string = if active { "yes" } else { "no" }`.

Add modernization rules with each language feature. Enable rules only when
the compiler supports their replacement. Preserve behavior and public APIs.
