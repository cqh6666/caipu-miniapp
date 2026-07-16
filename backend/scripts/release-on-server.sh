#!/usr/bin/env bash
set -euo pipefail

BACKEND_DIR="${BACKEND_DIR:-$(cd "$(dirname "$0")/.." && pwd)}"
RELEASES_DIR="${RELEASES_DIR:-${BACKEND_DIR}/releases}"
CURRENT_LINK="${CURRENT_LINK:-${BACKEND_DIR}/current}"
DATA_DIR="${DATA_DIR:-${BACKEND_DIR}/data}"
SQLITE_PATH="${SQLITE_PATH:-${DATA_DIR}/app.db}"
UPLOAD_DIR="${UPLOAD_DIR:-${DATA_DIR}/uploads}"
APP_ENV_FILE="${APP_ENV_FILE:-${BACKEND_DIR}/configs/prod.env}"
BACKUP_ROOT="${BACKUP_ROOT:-${BACKEND_DIR}/backups}"
BACKUP_ENV_FILE="${BACKUP_ENV_FILE:-/etc/${SERVICE_NAME:-caipu-backend}-backup.env}"
SERVICE_NAME="${SERVICE_NAME:-caipu-backend}"
SYSTEMCTL_BIN="${SYSTEMCTL_BIN:-systemctl}"
CURL_BIN="${CURL_BIN:-curl}"
APP_PORT="${APP_PORT:-8080}"
READINESS_PATH="${READINESS_PATH:-/readyz}"
READY_ATTEMPTS="${READY_ATTEMPTS:-30}"
READY_CONSECUTIVE_SUCCESSES="${READY_CONSECUTIVE_SUCCESSES:-3}"
READY_INTERVAL_SECONDS="${READY_INTERVAL_SECONDS:-2}"
RUN_BACKEND_TESTS="${RUN_BACKEND_TESTS:-1}"
VERIFY_SYSTEMD_CURRENT="${VERIFY_SYSTEMD_CURRENT:-1}"
RELEASE_RETENTION_COUNT="${RELEASE_RETENTION_COUNT:-5}"
BUILD_NICE="${BUILD_NICE:-10}"
GO_BUILD_GOMAXPROCS="${GO_BUILD_GOMAXPROCS:-1}"
GOCACHE_DIR="${GOCACHE_DIR:-/tmp/caipu-go-build-cache}"

log() {
  echo "==> $*"
}

fail() {
  echo "backend release failed: $*" >&2
  exit 1
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

run_low_priority() {
  if has_cmd ionice; then
    ionice -c3 nice -n "$BUILD_NICE" "$@"
    return
  fi
  nice -n "$BUILD_NICE" "$@"
}

sha256_file() {
  if has_cmd sha256sum; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  if has_cmd shasum; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi
  fail "sha256sum or shasum is required"
}

atomic_switch() {
  local target="$1"
  local next_link="${CURRENT_LINK}.next.$$"
  ln -s "$target" "$next_link"
  if mv -Tf "$next_link" "$CURRENT_LINK" 2>/dev/null; then
    return
  fi
  # BSD mv (used by local release tests) has no -T; -h keeps a symlink
  # destination from being followed. Production Ubuntu takes the branch above.
  mv -fh "$next_link" "$CURRENT_LINK"
}

load_backup_environment() {
  [[ -f "$BACKUP_ENV_FILE" ]] || return 0
  local key value
  while IFS='=' read -r key value; do
    key="${key//[[:space:]]/}"
    [[ -n "$key" && "$key" != \#* ]] || continue
    case "$key" in
      RETENTION_DAYS|OFFSITE_BACKUP_TARGET|REQUIRE_OFFSITE_BACKUP|RSYNC_BIN|RSYNC_RSH)
        if [[ "$value" == \"*\" && "$value" == *\" ]]; then
          value="${value:1:${#value}-2}"
        elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
          value="${value:1:${#value}-2}"
        fi
        export "$key=$value"
        ;;
      *)
        fail "unsupported key in backup environment file: $key"
        ;;
    esac
  done <"$BACKUP_ENV_FILE"
}

release_id_from_dir() {
  local release_dir="$1"
  awk -F= '$1 == "release_id" { print substr($0, index($0, "=") + 1); exit }' \
    "$release_dir/manifest.env" 2>/dev/null || true
}

wait_for_release() {
  local expected_release="$1"
  local consecutive=0
  local attempt observed headers
  headers="$(mktemp "${TMPDIR:-/tmp}/caipu-ready-headers.XXXXXX")"

  for ((attempt = 1; attempt <= READY_ATTEMPTS; attempt++)); do
    : >"$headers"
    if "$CURL_BIN" --fail --silent --show-error --max-time 3 \
      --dump-header "$headers" --output /dev/null \
      "http://127.0.0.1:${APP_PORT}${READINESS_PATH}" 2>/dev/null; then
      observed="$(awk 'tolower($1) == "x-release-id:" { gsub("\\r", "", $2); print $2 }' "$headers" | tail -n 1)"
      if [[ "$observed" == "$expected_release" ]]; then
        consecutive=$((consecutive + 1))
        if (( consecutive >= READY_CONSECUTIVE_SUCCESSES )); then
          rm -f -- "$headers"
          return 0
        fi
      else
        consecutive=0
      fi
    else
      consecutive=0
    fi
    if (( attempt < READY_ATTEMPTS )); then
      sleep "$READY_INTERVAL_SECONDS"
    fi
  done

  rm -f -- "$headers"
  return 1
}

rollback_binary() {
  local previous_target="$1"
  if [[ -n "$previous_target" && -x "$previous_target/server" ]]; then
    local previous_release_id
    previous_release_id="$(release_id_from_dir "$previous_target")"
    log "readiness failed; restoring previous binary: $previous_target"
    atomic_switch "$previous_target"
    if ! "$SYSTEMCTL_BIN" restart "$SERVICE_NAME"; then
      echo "Previous binary was selected but its service restart failed; inspect systemd immediately." >&2
      return
    fi
    if [[ -n "$previous_release_id" ]] && wait_for_release "$previous_release_id"; then
      echo "Previous binary restored. Database migrations were not reversed; review forward compatibility before any manual SQL rollback." >&2
    else
      echo "Previous binary was selected but did not become ready; inspect service logs immediately." >&2
    fi
    return
  fi

  echo "No previous release exists; stopping the unhealthy first release." >&2
  "$SYSTEMCTL_BIN" stop "$SERVICE_NAME" || true
  rm -f -- "$CURRENT_LINK"
}

prune_releases() {
  local keep="$RELEASE_RETENTION_COUNT"
  local index=0 release_dir current_target
  current_target="$(readlink -f "$CURRENT_LINK" 2>/dev/null || true)"

  while IFS= read -r release_dir; do
    [[ -n "$release_dir" ]] || continue
    index=$((index + 1))
    if (( index <= keep )) || [[ "$release_dir" == "$current_target" ]]; then
      continue
    fi
    rm -rf -- "$release_dir"
  done < <(find "$RELEASES_DIR" -mindepth 1 -maxdepth 1 -type d ! -name '.*' -print0 | \
    xargs -0 ls -1dt 2>/dev/null || true)
}

for value in "$READY_ATTEMPTS" "$READY_CONSECUTIVE_SUCCESSES" "$RELEASE_RETENTION_COUNT"; do
  [[ "$value" =~ ^[1-9][0-9]*$ ]] || fail "readiness attempts, consecutive successes, and retention count must be positive integers"
done
[[ "$READY_INTERVAL_SECONDS" =~ ^[0-9]+([.][0-9]+)?$ ]] || fail "READY_INTERVAL_SECONDS must be non-negative"
[[ "$RUN_BACKEND_TESTS" == "0" || "$RUN_BACKEND_TESTS" == "1" ]] || fail "RUN_BACKEND_TESTS must be 0 or 1"
[[ "$VERIFY_SYSTEMD_CURRENT" == "0" || "$VERIFY_SYSTEMD_CURRENT" == "1" ]] || fail "VERIFY_SYSTEMD_CURRENT must be 0 or 1"
[[ -d "$BACKEND_DIR/migrations" ]] || fail "migration directory not found: $BACKEND_DIR/migrations"
[[ -f "$APP_ENV_FILE" ]] || fail "production environment file not found: $APP_ENV_FILE"
[[ -f "$SQLITE_PATH" ]] || fail "SQLite database not found: $SQLITE_PATH"
[[ -d "$UPLOAD_DIR" ]] || fail "upload directory not found: $UPLOAD_DIR"
has_cmd go || fail "go is required"
has_cmd "$SYSTEMCTL_BIN" || fail "systemctl command is required: $SYSTEMCTL_BIN"
has_cmd "$CURL_BIN" || fail "curl command is required: $CURL_BIN"
load_backup_environment

if [[ "$VERIFY_SYSTEMD_CURRENT" == "1" ]]; then
  unit_exec_start="$($SYSTEMCTL_BIN show "$SERVICE_NAME" --property=ExecStart --value)"
  if [[ "$unit_exec_start" != *"${CURRENT_LINK}/server"* ]]; then
    fail "systemd ExecStart must point to ${CURRENT_LINK}/server; rerun bootstrap-server.sh before releasing"
  fi
fi

git_commit="$(git -C "$BACKEND_DIR" rev-parse HEAD)"
release_id="${RELEASE_ID:-$(date -u +%Y%m%dT%H%M%SZ)-${git_commit:0:12}}"
build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
go_toolchain="$(cd "$BACKEND_DIR" && go env GOVERSION)"
if [[ ! "$release_id" =~ ^[A-Za-z0-9._-]+$ ]]; then
  fail "RELEASE_ID may only contain letters, numbers, dot, underscore, and hyphen"
fi

mkdir -p "$RELEASES_DIR" "$BACKUP_ROOT"
staging_dir="${RELEASES_DIR}/.${release_id}.$$"
release_dir="${RELEASES_DIR}/${release_id}"
[[ ! -e "$release_dir" ]] || fail "release already exists: $release_dir"
mkdir "$staging_dir"
preflight_dir=""

cleanup() {
  [[ -z "${preflight_dir:-}" ]] || rm -rf -- "$preflight_dir"
  [[ -z "${staging_dir:-}" ]] || rm -rf -- "$staging_dir"
}
trap cleanup EXIT

cd "$BACKEND_DIR"
if [[ "$RUN_BACKEND_TESTS" == "1" ]]; then
  log "running backend tests before release"
  run_low_priority env GOMAXPROCS="$GO_BUILD_GOMAXPROCS" GOCACHE="$GOCACHE_DIR" \
    go test ./... -count=1
fi

log "building release $release_id"
run_low_priority env GOMAXPROCS="$GO_BUILD_GOMAXPROCS" GOCACHE="$GOCACHE_DIR" \
  go build -trimpath \
  -ldflags "-s -w -X github.com/cqh6666/caipu-miniapp/backend/internal/buildinfo.ReleaseID=${release_id} -X github.com/cqh6666/caipu-miniapp/backend/internal/buildinfo.GitCommit=${git_commit} -X github.com/cqh6666/caipu-miniapp/backend/internal/buildinfo.BuildTime=${build_time} -X github.com/cqh6666/caipu-miniapp/backend/internal/buildinfo.GoToolchain=${go_toolchain}" \
  -o "$staging_dir/server" ./cmd/server
cp -R "$BACKEND_DIR/migrations" "$staging_dir/migrations"
chmod 755 "$staging_dir" "$staging_dir/server" "$staging_dir/migrations"
find "$staging_dir/migrations" -type f -exec chmod 644 {} +

migration_manifest="$staging_dir/migrations.sha256"
while IFS= read -r migration_file; do
  migration_name="${migration_file#"$staging_dir/"}"
  printf '%s  %s\n' "$(sha256_file "$migration_file")" "$migration_name"
done < <(find "$staging_dir/migrations" -maxdepth 1 -type f -name '*.sql' -print | LC_ALL=C sort) >"$migration_manifest"
migration_count="$(wc -l <"$migration_manifest" | tr -d '[:space:]')"
migration_set_sha256="$(sha256_file "$migration_manifest")"
chmod 644 "$migration_manifest"

log "validating production configuration"
env APP_ENV_FILE="$APP_ENV_FILE" SQLITE_PATH="$SQLITE_PATH" \
  UPLOAD_DIR="$UPLOAD_DIR" MIGRATION_DIR="$staging_dir/migrations" \
  "$staging_dir/server" -check-config

log "creating consistent pre-release backup"
backup_output="$(
  SQLITE_PATH="$SQLITE_PATH" \
  UPLOAD_DIR="$UPLOAD_DIR" \
  BACKUP_ROOT="$BACKUP_ROOT" \
  RELEASE_ID="$release_id" \
    bash "$BACKEND_DIR/scripts/backup.sh"
)"
echo "$backup_output"
backup_dir="${backup_output##*Backup created: }"
[[ -d "$backup_dir" ]] || fail "backup script did not return a usable backup directory"

log "preflighting migrations against the consistent backup"
preflight_dir="$(mktemp -d "${TMPDIR:-/tmp}/caipu-migration-preflight.XXXXXX")"
cp "$backup_dir/app.db" "$preflight_dir/app.db"
mkdir "$preflight_dir/uploads"
env APP_ENV_FILE="$APP_ENV_FILE" SQLITE_PATH="$preflight_dir/app.db" \
  UPLOAD_DIR="$preflight_dir/uploads" MIGRATION_DIR="$staging_dir/migrations" \
  "$staging_dir/server" -migrate-only
rm -rf -- "$preflight_dir"
preflight_dir=""

binary_sha256="$(sha256_file "$staging_dir/server")"
cat >"$staging_dir/manifest.env" <<EOF
release_id=${release_id}
git_commit=${git_commit}
built_at=${build_time}
go_toolchain=${go_toolchain}
binary_sha256=${binary_sha256}
migration_count=${migration_count}
migration_set_sha256=${migration_set_sha256}
backup_dir=${backup_dir}
EOF
chmod 644 "$staging_dir/manifest.env"

log "applying forward migrations to the production database"
env APP_ENV_FILE="$APP_ENV_FILE" SQLITE_PATH="$SQLITE_PATH" \
  UPLOAD_DIR="$UPLOAD_DIR" MIGRATION_DIR="$staging_dir/migrations" \
  "$staging_dir/server" -migrate-only

mv "$staging_dir" "$release_dir"
staging_dir=""
previous_target="$(readlink -f "$CURRENT_LINK" 2>/dev/null || true)"
atomic_switch "$release_dir"

log "restarting $SERVICE_NAME"
if ! "$SYSTEMCTL_BIN" restart "$SERVICE_NAME"; then
  rollback_binary "$previous_target"
  fail "service restart failed; previous binary restoration was attempted"
fi

log "waiting for ${READY_CONSECUTIVE_SUCCESSES} consecutive readiness checks for $release_id"
if ! wait_for_release "$release_id"; then
  rollback_binary "$previous_target"
  fail "release did not become ready; previous binary restoration was attempted"
fi

prune_releases
log "backend release completed: $release_dir"
