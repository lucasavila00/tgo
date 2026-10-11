# Compile in the original module

## Decision and reason

Use each final generated Go path for both disk stages: original TGo for source
checking, then ordinary Go. Compile at the original module paths. `Compile(dir)`
returns nothing. There is no module copy, separate temporary source file, or
output-directory option.

Keeping original paths lets Go find embedded assets, cgo headers, vendor files,
and local dependencies. Relative replacements retain their meaning. Read owned
source without loading unrelated asset contents. Nested modules and data
directories are outside compiler ownership.

## Consequences

TGo inputs control output. Replace selected outputs and remove owned outputs
excluded from the build. Preserve original TGo, ordinary Go, and assets. Normal
Go commands can update module metadata.

A failed compilation can leave staged source or completed output at generated
paths. The next compilation replaces those paths from the inputs. See the
[driver package](../../internal/driver/readme.md) for the current steps and
[measurements](../performance.md) for the earlier copy comparison.

## Validation

Check TGo-only and mixed packages, subdirectory arguments, imports, build tags,
platform suffixes, and test variants. Replace invalid stale snapshots; remove
excluded owned outputs. Leave invalid inputs and snapshots outside ownership
unchanged.

Check embeds, cgo headers, vendor and workspace dependencies, relative
replacements, module updates, repeated compilation, and recovery after failure.
Compare written files with committed snapshots read before fixture compilation.
Run independent runtime references; see the [test guide](../../tests/readme.md).

Measure several packages and large assets with the same toolchain and caches.
Require one metadata load and one final Go load. Do not read unrelated asset
contents or write assets.
