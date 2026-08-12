#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"

deploy_output="$(bash "$ROOT_DIR/backend/scripts/deploy.sh" 2>&1 || true)"
for expected in '安全 tombstone' '不得删除此路径' 'scripts/deploy-backend-on-server.sh'; do
  if ! grep -Fq -- "$expected" <<< "$deploy_output"; then
    echo "deploy tombstone missing: $expected" >&2
    exit 1
  fi
done

if ! grep -Fq 'Frozen research artifact: unsupported' "$ROOT_DIR/scripts/probe-meituan-place-link.mjs"; then
  echo "Meituan research probe is not marked frozen/unsupported" >&2
  exit 1
fi

for expected in 'Archived / 非权威历史设计' 'backend/README.md' 'docs/backend-deploy-quickstart.md'; do
  if ! grep -Fq -- "$expected" "$ROOT_DIR/README-go.md"; then
    echo "archived Go design warning missing: $expected" >&2
    exit 1
  fi
done

echo "conditional artifact safety markers test passed"
