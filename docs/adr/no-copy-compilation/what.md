# Compile in the original module

## Decision

Compile at the original module paths. Write original TGo at each final
`_tgo.go` path for source checking, then replace selected files with ordinary
Go. Remove generated paths for excluded inputs. Stock Go checks the final files.

`Compile(dir)` returns nothing. There is no module copy, temporary source file,
or output-directory option.

## Why and consequences

Assets, cgo headers, vendor files, and local dependencies stay where Go expects
them. The driver reads owned source, without loading unrelated asset contents.
Nested modules and data directories are not compiler outputs.

TGo inputs control generated files. Ordinary Go and original TGo remain
unchanged. Normal Go commands can update module metadata. A failed compilation
can leave a source-checking version or completed output at generated paths;
the next compilation replaces those paths from the inputs.

[HOW](how.md) describes the disk stages. [PROOF](proof.md) defines their checks.
