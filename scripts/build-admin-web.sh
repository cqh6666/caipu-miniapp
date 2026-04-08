#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ADMIN_DIR="${ADMIN_DIR:-$ROOT_DIR/admin-web}"

if [[ ! -d "$ADMIN_DIR" ]]; then
  echo "admin-web directory not found: $ADMIN_DIR" >&2
  exit 1
fi

cd "$ADMIN_DIR"

if [[ ! -d node_modules ]]; then
  npm install
fi

npm run build
