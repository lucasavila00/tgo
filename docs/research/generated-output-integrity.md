# Generated output integrity

## Previous design

The `//tgo:v1` line contained a SHA-256 digest of these bytes:

1. The complete TGo source file.
2. The separator `\x00tgo generated body\x00`.
3. The complete formatted Go body.

This digest detected an accidental source change or generated-file edit. It was not an
authentication code. A person who changed the file could calculate a new digest. The declaration
verifier provided a second check for generated model APIs.

The digest also caused merge conflicts. Any source-byte change changed line 2, even when the
formatted Go output stayed the same. Any generated-body change also changed line 2. Thus, two
branches that changed separate parts of one source file both changed the same metadata line.

## Current design

The `//tgo:v2` line contains only the quoted source base name. It changes only when the metadata
version or source name changes.

Before `tgolint` exports model facts, it uses the compiler analysis that the source checks already
need. The analysis formats the expected output for every active TGo source. The linter requires the
complete file bytes to equal that output. It then runs the model declaration checks.

This comparison detects a stale source, a stale compiler output, a changed metadata line, and an
edit in any generated declaration or function. Generated-file ownership and stale-output removal
still use the fixed first line and output path. They do not depend on the former digest.

The check does not add a second package compilation. It moves the existing analysis before model
fact export and reuses its output. A source edit that produces the same formatted Go file causes no
generated diff. A source or compiler change that affects output changes only the affected Go text.

Sidecar metadata would move the common conflict to another tracked file. A narrower digest would
not protect the complete output. Exact compiler recomputation keeps the full check and removes the
changing digest.
