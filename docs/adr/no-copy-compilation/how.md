# HOW

1. Read module files. Record TGo inputs and their generated paths. Exclude
   matching stale generated files from the returned input set.
2. Write each original TGo input at its generated path in the module. Run
   `go/packages` there to select files, dependencies, exports, and test variants.
3. Have the private checker read these staged source files through the frontend.
   Check each package identity once and keep its checked types and positions.
4. Lower selected source packages. Write generated Go only to paths owned by TGo
   inputs. Remove generated paths for excluded inputs.
5. Run stock Go checking against the final files on disk. Collect selected
   generated files, original inputs, assets, and current module metadata.
   Make relative local replacement paths absolute in returned `go.mod`.

Go performs normal module loading and metadata updates in the source directory.
Keep filename suffixes and build constraints. The caller can write the returned
files to another directory. The compiler stages and replaces generated files;
Go reads ordinary Go, embed assets, and cgo headers at their original paths.

[WHAT](what.md) records the decision. [PROOF](proof.md) defines the checks.
