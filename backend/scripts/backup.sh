#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

timestamp="$(date +%Y%m%d-%H%M%S)"
mkdir -p backups

if [[ -f data/app.db ]]; then
  cp data/app.db "backups/app-${timestamp}.db"
fi

if [[ -d data/uploads ]]; then
  tar -czf "backups/uploads-${timestamp}.tar.gz" data/uploads
fi
