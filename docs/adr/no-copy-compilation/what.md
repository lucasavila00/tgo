# WHAT

## Decision

Compile against the original module directory without copying the module into
a temporary directory. Use overlays to present TGo files at generated Go paths
for package selection and to check the final generated Go.

Keep the private source checker and expression lowerer from the
[source-checking decision](../source-checking/what.md). This changes file
preparation and output handling, not TGo language rules.

## Reason

The driver currently reads module files into memory, writes them into a separate
directory, writes lowered Go there, and reads that output back. Assets and other
files are copied even when source checking does not need their contents. Moving
the module also requires adjusting relative replacement paths for checking.

The checker needs selected source, types, and operation records. It does not
need a complete module copy. Go's build tools can use alternative file contents
at original paths through an overlay. Ordinary Go files, embedded assets,
headers, and module dependencies can remain at their original locations.

Use two views: source for checking TGo, then generated Go for stock checking.
Keep both views separate from files visible to ordinary source-directory users.
Writing temporary stubs directly at generated paths would change the module
during compilation and would require restoring existing snapshots on failure.

## Scope and consequences

Preserve `Compile(dir) map[string][]byte`, compilation from subdirectories,
module-relative output paths, generated names, and file selection. Compilation
must not change the source directory. The caller still writes the returned
contents. Matching generated snapshots cannot affect checking.

The result still contains original TGo and other module files. That contract
requires reading those files; this decision removes their temporary disk
copies. Keep the returned module's replacement paths valid when the caller
writes it elsewhere. Do not change the public API to return only generated files.

Temporary overlay backing files are permitted outside the source directory.
They contain only alternative file contents required by the Go tools. This
decision does not require zero disk writes or a new cache.

Preserve valid TGo behavior and the
[automatic modernization rules](../../problem/README.md#adoption-and-automatic-modernization).
Measure compilation cost before claiming a speed improvement.

[HOW](how.md) defines the implementation. [PROOF](proof.md) defines acceptance.
