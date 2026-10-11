# WHAT

Compile in the original module directory without copying it. The current copy
duplicates assets and source that checking can read at their original paths.

An [overlay](../../guide/glossary.md#overlay) maps a file path to alternative
contents. Go sees those contents without changing the file on disk. Use two overlays: TGo source at generated
Go paths for package selection, then generated Go for final checking. Ordinary
files stay in place. Write only required overlay contents to temporary files
outside the source directory.

Keep the [private checker and lowerer](../source-checking/what.md), TGo behavior,
and `Compile(dir) map[string][]byte`. The result still includes source and
assets, so reading them remains necessary. The caller writes output.

Compilation must leave original files unchanged. Stale snapshots cannot affect
checking. Preserve file selection, output paths, and local replacements when
output is written elsewhere. Measure cost before claiming a speed improvement.

[HOW](how.md) defines the stages. [PROOF](proof.md) defines acceptance.
