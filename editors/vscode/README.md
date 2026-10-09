# TGo Navigation

This extension adds TGo syntax highlighting, symbol hover information, go to definition,
find references, document symbols, and workspace symbols to desktop VS Code.

On macOS, Linux, or another Unix system, run `./vscode.sh` from the TGo repository.
The script builds the `tgonav` helper for the current host, puts it in a private
VSIX, and installs that VSIX with the `code` command.

Set `tgo.navigation.helperPath` only when the extension must use a different
helper. The path must be absolute.

The extension reads saved `.tgo` files in an open workspace. It also reads the Go
files and module files that provide their type information. It does not edit source,
format code, or report diagnostics.

The extension works in local and desktop remote workspaces. Run the install
script in the environment that runs the workspace extension host. The extension
does not run in browser-only VS Code.
