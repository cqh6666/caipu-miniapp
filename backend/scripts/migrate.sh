#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
APP_ENV_FILE=configs/local.env go run ./cmd/server -migrate-only
