# TGo Navigation

This extension adds TGo syntax highlighting, go to definition, find references,
document symbols, and workspace symbols to desktop VS Code.

Install the matching `tgonav` helper on the system that runs the VS Code extension
host. Put it on `PATH`, or set `tgo.navigation.helperPath` to its absolute path.

The extension reads saved `.tgo` files in an open workspace. It also reads the Go
files and module files that provide their type information. It does not edit source,
format code, or report diagnostics.

The extension works in local and desktop remote workspaces. It does not run in
browser-only VS Code.
