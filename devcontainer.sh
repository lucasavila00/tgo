#!/bin/sh
set -eu

repository=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

: "${SSH_AUTH_SOCK:?start an SSH agent before the development container}"
if [ ! -S "$SSH_AUTH_SOCK" ]; then
	echo "devcontainer: SSH_AUTH_SOCK is not a socket" >&2
	exit 1
fi
if [ ! -f "$HOME/.ssh/known_hosts" ]; then
	echo "devcontainer: $HOME/.ssh/known_hosts does not exist" >&2
	exit 1
fi

devcontainer up --workspace-folder "$repository"

exec docker exec -it \
	--env SSH_AUTH_SOCK=/tmp/ssh-agent \
	--workdir /workspaces/tgo \
	tgo-devcontainer byobu
