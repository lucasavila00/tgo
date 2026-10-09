# Navigate TGo in VS Code

The TGo extension provides syntax highlighting, go to definition, find references, document
symbols, and workspace symbols. It does not edit source or report diagnostics.

Build the native helper and package the extension:

```sh
go build -o bin/tgonav ./cmd/tgonav
npm ci --prefix editors/vscode
npm run --prefix editors/vscode package
code --install-extension editors/vscode/tgo-navigation.vsix
```

Set the helper path in the workspace settings. Use an absolute path:

```json
{
  "tgo.navigation.helperPath": "/path/to/tgo/bin/tgonav"
}
```

For extension development, start an Extension Development Host from the repository root:

```sh
code --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD"
```

The extension updates its index after changes to `.tgo`, `.go`, `go.mod`, or `go.work` files. It
uses active Go build constraints and target file suffixes.

The extension runs in desktop VS Code and in desktop remote workspaces such as SSH and
containers. In a remote workspace, build `tgonav` on the remote system and use its remote path.
The extension does not run in browser-only VS Code.
