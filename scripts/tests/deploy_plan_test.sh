#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"

common_env=(
  RUN_GIT_PULL=0
  PLAN_ONLY=1
  HOST_RESOURCES_CPU_COUNT=2
  HOST_RESOURCES_MEM_TOTAL_MB=2048
  HOST_RESOURCES_SWAP_TOTAL_MB=0
)

admin_plan="$(env "${common_env[@]}" DEPLOY_SCOPE=admin-web bash "$ROOT_DIR/scripts/deploy-on-server.sh")"
for expected in '- build admin-web: yes' '- cpu: 2 vCPU' '- memory: 2048 MiB' '- swap: 0 MiB'; do
  if ! grep -Fq -- "$expected" <<< "$admin_plan"; then
    echo "admin PLAN_ONLY output missing: $expected" >&2
    exit 1
  fi
done

backend_plan="$(env "${common_env[@]}" DEPLOY_SCOPE=backend bash "$ROOT_DIR/scripts/deploy-on-server.sh")"
if ! grep -Fq -- '- run backend tests on server: no' <<< "$backend_plan"; then
  echo "backend PLAN_ONLY must skip remote tests by default" >&2
  exit 1
fi

backend_test_plan="$(env "${common_env[@]}" DEPLOY_SCOPE=backend RUN_BACKEND_TESTS=1 bash "$ROOT_DIR/scripts/deploy-on-server.sh")"
if ! grep -Fq -- '- run backend tests on server: yes' <<< "$backend_test_plan"; then
  echo "backend PLAN_ONLY did not preserve explicit remote test opt-in" >&2
  exit 1
fi

sidecar_plan="$(env "${common_env[@]}" SIDECAR_INSTALL_MODE=always bash "$ROOT_DIR/scripts/deploy-linkparse-sidecar-on-server.sh")"
for expected in '- install sidecar deps: yes' '- restart sidecar: yes' '- cpu: 2 vCPU'; do
  if ! grep -Fq -- "$expected" <<< "$sidecar_plan"; then
    echo "sidecar PLAN_ONLY output missing: $expected" >&2
    exit 1
  fi
done

echo "deploy PLAN_ONLY contract test passed"
