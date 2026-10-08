#!/bin/sh
set -eu

repository=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

devcontainer up --workspace-folder "$repository"
exec docker exec -it go2-devcontainer byobu
