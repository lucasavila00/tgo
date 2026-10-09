# Navigate TGo in VS Code

The TGo extension provides syntax highlighting, go to definition, find references, document
symbols, and workspace symbols. It does not edit source or report diagnostics.

Build the native helper and install the extension dependencies:

```sh
go build -o bin/tgonav ./cmd/tgonav
npm ci --prefix editors/vscode
```

Set the helper path in the workspace settings. Use an absolute path:

```json
{
  "tgo.navigation.helperPath": "/path/to/tgo/bin/tgonav"
}
```

Start an Extension Development Host from the repository root:

```sh
code --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD"
```

The extension updates its index after changes to `.tgo`, `.go`, `go.mod`, or `go.work` files. It
uses active Go build constraints and target file suffixes.

The extension runs in desktop VS Code and in desktop remote workspaces such as SSH and
containers. In a remote workspace, build `tgonav` on the remote system and use its remote path.
The extension does not run in browser-only VS Code.
