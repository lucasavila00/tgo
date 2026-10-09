# Navigate TGo in VS Code

The TGo extension provides syntax highlighting, go to definition, find references, document
symbols, and workspace symbols. It does not edit source or report diagnostics.

Build and install the private extension for the current host:

```sh
./vscode.sh
```

The script builds `tgonav`, includes it in the VSIX, and installs the VSIX with
the `code` command. Set `TGO_VSCODE_CODE` when that command has another name.

The bundled helper is the default. To use a different build, set its absolute
path in the workspace settings:

```json
{
  "tgo.navigation.helperPath": "/path/to/tgo/bin/tgonav"
}
```

To package without installation, or before extension development, run:

```sh
./vscode.sh --package-only
```

Then start an Extension Development Host from the repository root:

```sh
code --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD"
```

The extension updates its index after changes to `.tgo`, `.go`, `go.mod`, or `go.work` files. It
uses active Go build constraints and target file suffixes.

Navigation reads saved source. Save a changed document before you request a definition,
references, or symbols.

The extension runs in desktop VS Code and in desktop remote workspaces such as SSH and
containers. In a remote workspace, run `vscode.sh` in the environment that runs the workspace
extension host. The extension does not run in browser-only VS Code.
