# WHAT

Compile directly in the original module. Keep the private source checker,
lowerer, and `Compile(dir) map[string][]byte`.

Each `name.tgo` owns its `name_tgo.go` output. First write original TGo at that
path so Go can select packages, imports, build constraints, and test variants.
Check the TGo source. Then replace selected files with generated Go and remove
generated paths for excluded inputs. Stock Go checks these final files.

Compilation updates generated files on disk. Go can update `go.mod` and
`go.sum` through its normal module commands. Original TGo, ordinary Go, assets,
headers, and dependencies stay at their existing paths.

Return generated Go with original source, assets, and current module metadata.
Keep output names and adjust returned local replacement paths for an output
directory elsewhere. A failed compilation can leave source-checking versions at
generated paths. The next compilation replaces them from TGo inputs.

[HOW](how.md) defines the stages. [PROOF](proof.md) defines the checks.
