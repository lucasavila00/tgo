# Format TGo source

Use `tgofmt` to format `.tgo` source. With no file argument, it reads standard input and writes
the result to standard output.

```sh
tgofmt < input.tgo
tgofmt file.tgo
tgofmt -w file.tgo
tgofmt -l file.tgo
```

`-w` replaces each named file and keeps its permission bits. It cannot write standard input.
`-l` lists each named file that would change. For standard input, it prints `<standard input>`
when the source would change.

The formatter keeps comments and build constraints. It gives the same output after a second run.
It reports malformed source and does not change a malformed named file.
