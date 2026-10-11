# Source checks

Run `make lint` before you submit code. Install golangci-lint v2.14.0 first:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
make lint
```

The checks cover unchecked errors, Go vet findings, Staticcheck bug checks,
unused code, documentation, basic names, and Go formatting. Revive supplies the
style checks that replace the archived golint tool.

Each function has a limit of 100 lines, including comments. Split a long function
into operations with clear inputs and results. Each tracked Go or TGo source file
has a limit of 700 lines, including comments and blank lines.

The file check includes fixture inputs and reference programs. Generated
`*_tgo.go` and `*_tgo_test.go` outputs are excluded from the file limit. Go lint
checks exclude `tests/testdata`, which contains generated outputs and independent
test programs. Test functions in the main test package remain checked.

Pull request CI runs `make lint` before the test suite.
