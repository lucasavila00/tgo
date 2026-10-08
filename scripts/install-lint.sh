#!/bin/sh
set -eu

version=v2.14.0
installer=$(mktemp)
trap 'rm -f "$installer"' EXIT
curl -sSfL https://golangci-lint.run/install.sh -o "$installer"
sh "$installer" -b "$(go env GOPATH)/bin" "$version"
