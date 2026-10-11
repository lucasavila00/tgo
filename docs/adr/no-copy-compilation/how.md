# HOW

1. Query Go packages from the original module root with
   [`go/packages` overlays](https://pkg.go.dev/golang.org/x/tools/go/packages#Config).
   Map each TGo input to its absolute generated Go path, including new paths.
   Preserve package declarations, imports, build constraints, suffixes, test
   variants, and dependency exports. Replace matching stale snapshots.
2. Have the private loader read original TGo from the input map and ordinary Go
   from disk. Parse TGo through the frontend; check actual source, not stubs.
   Retain positions and cache checked packages by package identity.
3. After source checking, lower selected packages into memory. Build the final
   overlay from generated Go. Hide matching snapshots for excluded TGo inputs.
   Run stock Go checking against this overlay.
4. Return generated contents with original TGo and other module files. Keep
   `.git`, `.worktrees`, and snapshot exclusions. Adjust returned replacement
   paths where needed for output written elsewhere; leave module files on disk
   unchanged.

Assets, headers, ordinary Go, and dependencies stay at original paths. Preserve
embed and supported cgo behavior. Remove temporary backing files on success or
failure. Write returned output before running generated programs.

[WHAT](what.md) records the decision. [PROOF](proof.md) defines the checks.
