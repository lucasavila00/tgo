#!/bin/sh

set -eu

if [ "${GITHUB_ACTIONS:-}" = true ] || [ "${TGO_LOCAL_VALIDATION_LOCK_HELD:-}" = 1 ]; then
	exec env TGO_LOCAL_VALIDATION_LOCK_HELD=1 "$@"
fi

common_dir=$(git rev-parse --git-common-dir)
case "$common_dir" in
	/*) ;;
	*) common_dir=$(cd "$common_dir" && pwd) ;;
esac

exec flock "$common_dir/tgo-local-validation.lock" \
	env TGO_LOCAL_VALIDATION_LOCK_HELD=1 "$@"
