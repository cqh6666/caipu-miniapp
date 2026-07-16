#!/usr/bin/env bash
set -euo pipefail

CURL_BIN="${CURL_BIN:-curl}"
APP_PORT="${APP_PORT:-8080}"
BACKEND_BASE_URL="${BACKEND_BASE_URL:-http://127.0.0.1:${APP_PORT}}"
LIVENESS_PATH="${LIVENESS_PATH:-/livez}"
READINESS_PATH="${READINESS_PATH:-/readyz}"
EXPECTED_LIVE_STATUS="${EXPECTED_LIVE_STATUS:-200}"
EXPECTED_READY_STATUS="${EXPECTED_READY_STATUS:-200}"
EXPECTED_RELEASE_ID="${EXPECTED_RELEASE_ID:-}"
REQUEST_TIMEOUT_SECONDS="${REQUEST_TIMEOUT_SECONDS:-5}"

usage() {
  cat <<'EOF'
Usage:
  bash backend/scripts/check-service-health.sh

Normal post-release verification:
  EXPECTED_RELEASE_ID=<release-id> bash backend/scripts/check-service-health.sh

Controlled readiness fault verification:
  EXPECTED_READY_STATUS=503 bash backend/scripts/check-service-health.sh

Environment variables:
  BACKEND_BASE_URL=http://127.0.0.1:8080
  LIVENESS_PATH=/livez
  READINESS_PATH=/readyz
  EXPECTED_LIVE_STATUS=200|any
  EXPECTED_READY_STATUS=200|503|any
  EXPECTED_RELEASE_ID=
  REQUEST_TIMEOUT_SECONDS=5
  CURL_BIN=curl
EOF
}

if [[ "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

fail() {
  echo "service health check failed: $*" >&2
  exit 1
}

validate_expected_status() {
  local name="$1" value="$2"
  if [[ "$value" != "any" && ! "$value" =~ ^[1-5][0-9][0-9]$ ]]; then
    fail "$name must be an HTTP status code or any"
  fi
}

command -v "$CURL_BIN" >/dev/null 2>&1 || fail "curl command is required: $CURL_BIN"
[[ "$BACKEND_BASE_URL" =~ ^https?:// ]] || fail "BACKEND_BASE_URL must be an absolute HTTP(S) URL"
[[ "$LIVENESS_PATH" == /* ]] || fail "LIVENESS_PATH must start with /"
[[ "$READINESS_PATH" == /* ]] || fail "READINESS_PATH must start with /"
[[ "$REQUEST_TIMEOUT_SECONDS" =~ ^[1-9][0-9]*$ ]] || fail "REQUEST_TIMEOUT_SECONDS must be a positive integer"
validate_expected_status "EXPECTED_LIVE_STATUS" "$EXPECTED_LIVE_STATUS"
validate_expected_status "EXPECTED_READY_STATUS" "$EXPECTED_READY_STATUS"
BACKEND_BASE_URL="${BACKEND_BASE_URL%/}"

tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/caipu-service-health.XXXXXX")"
cleanup() {
  rm -rf -- "$tmpdir"
}
trap cleanup EXIT

probe() {
  local name="$1" path="$2" expected_status="$3"
  local headers="$tmpdir/${name}.headers"
  local body="$tmpdir/${name}.body"
  local status release_id

  if ! status="$(
    "$CURL_BIN" --silent --show-error --max-time "$REQUEST_TIMEOUT_SECONDS" \
      --dump-header "$headers" --output "$body" --write-out '%{http_code}' \
      "${BACKEND_BASE_URL}${path}"
  )"; then
    fail "$name request could not reach ${BACKEND_BASE_URL}${path}"
  fi

  release_id="$(awk 'tolower($1) == "x-release-id:" { gsub("\\r", "", $2); print $2 }' "$headers" | tail -n 1)"
  printf '%s status=%s release_id=%s url=%s\n' \
    "$name" "$status" "${release_id:-missing}" "${BACKEND_BASE_URL}${path}"

  if [[ "$expected_status" != "any" && "$status" != "$expected_status" ]]; then
    fail "$name returned HTTP $status, expected $expected_status"
  fi
  if [[ -n "$EXPECTED_RELEASE_ID" && "$release_id" != "$EXPECTED_RELEASE_ID" ]]; then
    fail "$name release ID ${release_id:-missing} does not match $EXPECTED_RELEASE_ID"
  fi
}

probe "livez" "$LIVENESS_PATH" "$EXPECTED_LIVE_STATUS"
probe "readyz" "$READINESS_PATH" "$EXPECTED_READY_STATUS"
