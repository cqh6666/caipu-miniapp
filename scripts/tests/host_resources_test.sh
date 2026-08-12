#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
HELPER="$ROOT_DIR/scripts/lib/host-resources.sh"

source_output="$(bash -c 'source "$1"' _ "$HELPER")"
if [[ -n "$source_output" ]]; then
  echo "host resource helper produced output while sourced" >&2
  exit 1
fi

source "$HELPER"

export HOST_RESOURCES_CPU_COUNT=2
export HOST_RESOURCES_MEM_TOTAL_MB=2048
export HOST_RESOURCES_SWAP_TOTAL_MB=0

if ! host_resources_is_low_resource 4 3072 1024; then
  echo "low-resource fixture was not detected" >&2
  exit 1
fi

export HOST_RESOURCES_CPU_COUNT=8
export HOST_RESOURCES_MEM_TOTAL_MB=8192
export HOST_RESOURCES_SWAP_TOTAL_MB=2048

if host_resources_is_low_resource 4 3072 1024; then
  echo "healthy-resource fixture was classified as low-resource" >&2
  exit 1
fi

summary="$(host_resources_print_summary)"
for expected in '- cpu: 8 vCPU' '- memory: 8192 MiB' '- swap: 2048 MiB'; do
  if ! grep -Fq -- "$expected" <<< "$summary"; then
    echo "resource summary missing: $expected" >&2
    exit 1
  fi
done

echo "host resource helper test passed"
