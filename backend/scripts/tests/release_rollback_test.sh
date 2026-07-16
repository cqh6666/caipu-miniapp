#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/caipu-release-test.XXXXXX")"
BACKEND="$TEST_ROOT/backend"
FAKE_BIN="$TEST_ROOT/fake-bin"
CURRENT="$BACKEND/current"
SYSTEMCTL_LOG="$TEST_ROOT/systemctl.log"
CURL_LOG="$TEST_ROOT/curl.log"

cleanup() {
  rm -rf -- "$TEST_ROOT"
}
trap cleanup EXIT

mkdir -p "$BACKEND/scripts" "$BACKEND/migrations" "$BACKEND/configs" \
  "$BACKEND/data/uploads" "$BACKEND/releases/previous" "$FAKE_BIN"
cp "$SCRIPT_ROOT/backup.sh" "$BACKEND/scripts/backup.sh"
cp "$SCRIPT_ROOT/check-service-health.sh" "$BACKEND/scripts/check-service-health.sh"
printf '%s\n' 'CREATE TABLE fixture (id INTEGER PRIMARY KEY);' >"$BACKEND/migrations/001_fixture.sql"
printf '%s\n' 'APP_ENV=local' >"$BACKEND/configs/prod.env"
chmod 600 "$BACKEND/configs/prod.env"
printf '%s\n' 'upload fixture' >"$BACKEND/data/uploads/fixture.txt"
sqlite3 "$BACKEND/data/app.db" 'CREATE TABLE items (id INTEGER PRIMARY KEY); INSERT INTO items DEFAULT VALUES;'
cat >"$TEST_ROOT/backup.env" <<'EOF'
RETENTION_DAYS=7
REQUIRE_OFFSITE_BACKUP=0
EOF

cat >"$BACKEND/releases/previous/server" <<'SERVER'
#!/usr/bin/env bash
exit 0
SERVER
chmod 755 "$BACKEND/releases/previous/server"
mkdir "$BACKEND/releases/previous/migrations"
cat >"$BACKEND/releases/previous/manifest.env" <<'EOF'
release_id=previous
git_commit=previous
EOF
ln -s "$BACKEND/releases/previous" "$CURRENT"

git -C "$BACKEND" init -q
git -C "$BACKEND" config user.email release-test@example.com
git -C "$BACKEND" config user.name release-test
git -C "$BACKEND" add migrations configs
git -C "$BACKEND" commit -qm fixture

cat >"$FAKE_BIN/go" <<'FAKE_GO'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "test" ]]; then
  exit 0
fi
if [[ "${1:-}" == "env" && "${2:-}" == "GOVERSION" ]]; then
  if [[ "$(pwd -P)" != "$(cd "${EXPECTED_GO_ENV_DIR:?}" && pwd -P)" ]]; then
    echo "go env GOVERSION must run from the backend module: $PWD" >&2
    exit 1
  fi
  echo "go1.test"
  exit 0
fi
output=""
while (($#)); do
  if [[ "$1" == "-o" ]]; then
    output="$2"
    shift 2
    continue
  fi
  shift
done
[[ -n "$output" ]]
cat >"$output" <<'SERVER'
#!/usr/bin/env bash
case "${1:-}" in
  -check-config|-migrate-only) exit 0 ;;
  *) exit 0 ;;
esac
SERVER
chmod 755 "$output"
FAKE_GO
chmod 755 "$FAKE_BIN/go"

cat >"$FAKE_BIN/systemctl" <<EOF
#!/usr/bin/env bash
set -euo pipefail
case "\${1:-}" in
  show)
    echo "path=$CURRENT/server"
    ;;
  restart|stop)
    echo "\$1" >>"$SYSTEMCTL_LOG"
    ;;
esac
EOF
chmod 755 "$FAKE_BIN/systemctl"

cat >"$FAKE_BIN/curl" <<EOF
#!/usr/bin/env bash
set -euo pipefail
headers=""
output=""
write_out=""
url=""
while ((\$#)); do
  case "\$1" in
    --dump-header) headers="\$2"; shift 2 ;;
    --output) output="\$2"; shift 2 ;;
    --write-out) write_out="\$2"; shift 2 ;;
    --max-time) shift 2 ;;
    --fail|--silent|--show-error) shift ;;
    *) url="\$1"; shift ;;
  esac
done
printf '%s\n' "\$url" >>"$CURL_LOG"
target="\$(readlink -f "$CURRENT")"
release_id="\$(awk -F= '\$1 == "release_id" { print \$2; exit }' "\$target/manifest.env")"
if [[ "\$release_id" == "new-release" ]]; then
  exit 22
fi
printf 'HTTP/1.1 200 OK\r\nX-Release-ID: %s\r\n\r\n' "\$release_id" >"\$headers"
if [[ -n "\$output" && "\$output" != "/dev/null" ]]; then
  printf '{"status":"ok"}\n' >"\$output"
fi
if [[ -n "\$write_out" ]]; then
  printf '200'
fi
EOF
chmod 755 "$FAKE_BIN/curl"

set +e
PATH="$FAKE_BIN:$PATH" \
EXPECTED_GO_ENV_DIR="$BACKEND" \
BACKEND_DIR="$BACKEND" \
BACKUP_ENV_FILE="$TEST_ROOT/backup.env" \
APP_ENV_FILE="$BACKEND/configs/prod.env" \
SYSTEMCTL_BIN="$FAKE_BIN/systemctl" \
CURL_BIN="$FAKE_BIN/curl" \
RELEASE_ID="new-release" \
RUN_BACKEND_TESTS=0 \
READY_ATTEMPTS=1 \
READY_CONSECUTIVE_SUCCESSES=1 \
READY_INTERVAL_SECONDS=0 \
RELEASE_RETENTION_COUNT=2 \
  bash "$SCRIPT_ROOT/release-on-server.sh" >/dev/null 2>&1
status=$?
set -e

if [[ "$status" == "0" ]]; then
  echo "unhealthy release unexpectedly succeeded" >&2
  exit 1
fi
if [[ "$(readlink -f "$CURRENT")" != "$(readlink -f "$BACKEND/releases/previous")" ]]; then
  echo "current symlink was not restored to the previous release" >&2
  exit 1
fi
if [[ "$(grep -c '^restart$' "$SYSTEMCTL_LOG")" != "2" ]]; then
  echo "expected new-release restart and rollback restart" >&2
  exit 1
fi
if ! grep -q '/livez$' "$CURL_LOG" || ! grep -q '/readyz$' "$CURL_LOG"; then
  echo "rollback verification did not probe both liveness and readiness" >&2
  exit 1
fi
if [[ ! -d "$BACKEND/releases/new-release" ]]; then
  echo "failed release should remain available for diagnosis" >&2
  exit 1
fi
if [[ ! -s "$BACKEND/releases/new-release/migrations.sha256" ]]; then
  echo "release migration manifest is missing" >&2
  exit 1
fi
for key in git_commit built_at go_toolchain binary_sha256 migration_count migration_set_sha256; do
  if ! grep -q "^${key}=" "$BACKEND/releases/new-release/manifest.env"; then
    echo "release manifest is missing ${key}" >&2
    exit 1
  fi
done
if ! grep -q '^go_toolchain=go1.test$' "$BACKEND/releases/new-release/manifest.env"; then
  echo "release manifest did not use the backend module toolchain" >&2
  exit 1
fi
if [[ -z "$(find "$BACKEND/backups" -mindepth 1 -maxdepth 1 -type d -name 'backup-*' -print -quit)" ]]; then
  echo "pre-release backup was not created" >&2
  exit 1
fi

echo "release rollback integration test passed"
