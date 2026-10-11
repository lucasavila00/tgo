# Public transpiler API

This package exposes module compilation and statement insertion. Applications
import `github.com/lucasavila00/tgo/pkg/transpiler`.

`Compile(directory)` calls the internal driver. It returns a map from
module-relative paths to emitted file contents. Invalid source and compiler
failures cause a panic. The caller decides where to write the files.

`Load(directory, pattern)` loads one package of ordinary Go source. Its `Source`
contains the checked package and expression sites. Each site records its file,
byte offsets, and whether it has an evaluation in the current function.

`source.InsertAfter(site, callback)` calls the lowerer. The callback receives
references to the evaluated results and returns Go statements. For example,
it can insert a return before the parent expression uses those results.
The result maps absolute file paths to formatted Go. The original typed AST
stays available for another transformation. See the [API guide](../../docs/guide/transpiler.md)
for signatures and callback requirements.
