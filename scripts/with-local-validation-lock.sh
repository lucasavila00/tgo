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

exec 9>"$common_dir/tgo-local-validation.lock"
flock 9

meminfo=${TGO_LOCAL_VALIDATION_MEMINFO:-/proc/meminfo}
minimum_kb=${TGO_LOCAL_VALIDATION_MIN_AVAILABLE_KB:-6291456}
memory_max=${TGO_LOCAL_VALIDATION_MEMORY_MAX:-4G}
systemd_run=${TGO_LOCAL_VALIDATION_SYSTEMD_RUN:-systemd-run}

available_kb=$(awk '$1 == "MemAvailable:" { print $2; exit }' "$meminfo")
if [ -z "$available_kb" ]; then
	echo "local validation cannot read MemAvailable from $meminfo" >&2
	exit 1
fi
if [ "$available_kb" -lt "$minimum_kb" ]; then
	echo "local validation needs $minimum_kb kB available; only $available_kb kB is available" >&2
	exit 1
fi
if ! command -v "$systemd_run" >/dev/null 2>&1; then
	echo "local validation needs systemd-run to enforce its process-tree memory limit" >&2
	exit 1
fi

if "$systemd_run" --scope --quiet --collect --same-dir \
	--property="MemoryMax=$memory_max" --property=MemorySwapMax=0 \
	--property=MemoryOOMGroup=yes \
	--setenv=TGO_LOCAL_VALIDATION_LOCK_HELD=1 -- "$@"; then
	exit 0
else
	status=$?
	if [ "$status" -eq 137 ]; then
		echo "local validation was killed inside the $memory_max process-tree memory limit; check the systemd log for an out-of-memory kill" >&2
	else
		echo "local validation failed with status $status inside the $memory_max process-tree memory limit" >&2
	fi
	exit "$status"
fi
