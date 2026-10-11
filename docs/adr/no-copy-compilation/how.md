# Use the final generated paths for both stages

1. Find the module root. List file paths, exclude stale outputs, and read owned
   Go and TGo source. Nested `go.mod` files mark separate modules. Exclude
   vendor, testdata, and directories starting with `.` or `_` from ownership.
2. Write each TGo input at its generated Go path. Keep platform and test suffixes
   at the end. Ask `go/packages` for file selection and dependency metadata.
3. Check selected source through the frontend and private checker. Cache each
   package ID and keep types and source positions for lowering.
4. Lower checked expressions. Write ordinary Go only to owned generated paths.
   Remove owned paths for inputs excluded from this build.
5. Check the final module with stock Go. Return no file contents.

The compiler creates no separate temporary files or module. Ordinary Go,
embedded assets, headers, and dependencies stay at their original paths.
Relative local replacements stay relative. Go can update `go.mod` and `go.sum`
through normal module loading.

See [the complete process](../../arch/transpiler/readme.md) for examples and
file responsibilities. [WHAT](what.md) explains the decision; [PROOF](proof.md)
defines the checks.
