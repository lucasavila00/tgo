#!/bin/sh
set -eu

repository=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
extension="$repository/editors/vscode"
helper="$extension/bin/tgonav"
package_only=false

case "${1-}" in
  "") ;;
  --package-only) package_only=true ;;
  *)
    echo "usage: ./vscode.sh [--package-only]" >&2
    exit 2
    ;;
esac

mkdir -p "$extension/bin"
(cd "$repository" && go build -o "$helper" ./cmd/tgonav)
npm ci --prefix "$extension"
npm run --prefix "$extension" package

if [ "$package_only" = true ]; then
  exit 0
fi

code_command=${TGO_VSCODE_CODE:-code}
"$code_command" --install-extension "$extension/tgo-navigation.vsix" --force
