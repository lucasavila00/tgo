# Glossary

## Overlay

A mapping that tells Go's build tools to use alternative contents at a file
path without changing the original file. It can also present a new file or hide
an existing file during a build.

The Go command supports overlays through its
[`-overlay` flag](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies).
[`go/packages.Config.Overlay`](https://pkg.go.dev/golang.org/x/tools/go/packages#Config)
accepts a map from absolute paths to file contents.

For example, TGo can present generated `main_tgo.go` contents to Go while assets
stay at their original paths. This is a build-tool feature. It adds no Go syntax
and does not change files seen by the running program.
