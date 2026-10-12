#!/bin/sh
set -eu

# Tests can keep obsolete public APIs alive. Start from the command instead.
findings=$(go run golang.org/x/tools/cmd/deadcode \
    -filter '^github.com/lucasavila00/tgo/pkg/' ./cmd/tgo)
if [ -n "$findings" ]; then
    printf '%s\n' "$findings"
    exit 1
fi
