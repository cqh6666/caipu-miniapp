#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

cd "$ROOT_DIR"
export DEPLOY_SCOPE=backend

exec bash scripts/deploy-on-server.sh "$@"
