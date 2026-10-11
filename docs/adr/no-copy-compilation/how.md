# HOW

1. List module paths. Read owned Go and TGo source, excluding matching stale
   generated files. Record TGo inputs and their generated paths. A nested
   `go.mod` marks a separate module. Exclude nested modules, vendor dependencies,
   `testdata`, and directories whose names start with `.` or `_`. Do not read
   unrelated asset contents.
2. Write each original TGo input at its generated path in the module. Run
   `go/packages` there to select files, dependencies, exports, and test variants.
3. Have the private checker read these staged source files through the frontend.
   Check each package identity once and keep its checked types and positions.
4. Lower selected source packages. Write generated Go only to paths owned by TGo
   inputs. Remove generated paths for excluded inputs.
5. Run stock Go checking against the final files on disk. Return no file contents.

Go performs normal module loading and metadata updates in the source directory.
Keep filename suffixes and build constraints. The compiler stages and replaces
generated files. Go reads ordinary Go, embed assets, and cgo headers at their
original paths.

[WHAT](what.md) records the decision. [PROOF](proof.md) defines the checks.
