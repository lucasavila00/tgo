#!/bin/sh
set -eu

repository=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

devcontainer up --workspace-folder "$repository"
exec docker exec -it --workdir /workspaces/go2 go2-devcontainer byobu
