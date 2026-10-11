# TGo

TGo adds error propagation and value-producing `if` expressions to Go. The
compiler writes ordinary Go files beside the `.tgo` inputs. Go then builds and
runs those files.

```text
value := fetch()!
label := if value.Active { "active" } else { "inactive" }
```

Start with the [user guide](docs/guide/transpiler.md) for syntax and commands.
Read [how compilation works](docs/arch/transpiler/readme.md) for the complete
process, from source files to generated Go.

- [Source checking](docs/arch/transpiler/source-checking.md): how TGo expressions get their types.
- [Lowering](docs/arch/transpiler/correctness.md): how generated statements preserve execution.
- [Tests](docs/guide/testing.md): fixtures, references, and CI checks.
- [Language vision](docs/problem/vision.md): the intended language, including planned features.
