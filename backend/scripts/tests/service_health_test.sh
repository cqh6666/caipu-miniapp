#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/caipu-service-health-test.XXXXXX")"
FAKE_CURL="$TEST_ROOT/curl"

cleanup() {
  rm -rf -- "$TEST_ROOT"
}
trap cleanup EXIT

cat >"$FAKE_CURL" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail

headers=""
output=""
url=""
while (($#)); do
  case "$1" in
    --dump-header|--output|--write-out|--max-time)
      if [[ "$1" == "--dump-header" ]]; then headers="$2"; fi
      if [[ "$1" == "--output" ]]; then output="$2"; fi
      shift 2
      ;;
    --silent|--show-error)
      shift
      ;;
    *)
      url="$1"
      shift
      ;;
  esac
done

case "$url" in
  */livez) status="${LIVE_STATUS:-200}" ;;
  */readyz) status="${READY_STATUS:-200}" ;;
  *) status="404" ;;
esac
printf 'HTTP/1.1 %s Test\r\nX-Release-ID: %s\r\n\r\n' "$status" "${RELEASE_ID:-test-release}" >"$headers"
printf '{"status":%s}\n' "$status" >"$output"
printf '%s' "$status"
FAKE
chmod 755 "$FAKE_CURL"

run_health_check() {
  CURL_BIN="$FAKE_CURL" \
  EXPECTED_RELEASE_ID="test-release" \
    bash "$SCRIPT_ROOT/check-service-health.sh" "$@"
}

run_health_check >/dev/null

READY_STATUS=503 EXPECTED_READY_STATUS=503 run_health_check >/dev/null

set +e
READY_STATUS=503 run_health_check >/dev/null 2>&1
status=$?
set -e
if [[ "$status" == "0" ]]; then
  echo "unexpectedly accepted an unhealthy readiness response" >&2
  exit 1
fi

set +e
RELEASE_ID="other-release" run_health_check >/dev/null 2>&1
status=$?
set -e
if [[ "$status" == "0" ]]; then
  echo "unexpectedly accepted a mismatched release ID" >&2
  exit 1
fi

echo "service health integration test passed"
