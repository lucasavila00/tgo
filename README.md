# TGo

TGo adds error propagation and value-producing `if` expressions to Go. The
compiler writes ordinary Go files beside the `.tgo` inputs. Go then builds and
runs those files.

```text
value := fetch()!
label := if value.Active { "active" } else { "inactive" }
```

Start with the [user guide](docs/guide/transpiler.md) for syntax and commands.
Read the [driver package](internal/driver/readme.md) for the complete compiler
process and shared glossary. Package explanations come from their Go comments.

- [Frontend](internal/frontend/readme.md): parsing added syntax.
- [Checker](internal/checker/readme.md): checking source types and scope.
- [Lowering](internal/lowering/readme.md): preserving execution in generated Go.
- [Public API](pkg/transpiler/readme.md): compilation and expression insertion.
- [Tests](tests/readme.md): fixtures, references, and CI checks.
- [Performance](docs/performance.md): historical measurements.
- [Design decisions](docs/adr/): reasons, consequences, and validation plans.
- [Language vision](docs/problem/vision.md): the intended language, including planned features.
