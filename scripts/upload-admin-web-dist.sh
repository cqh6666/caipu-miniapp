#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ADMIN_DIR="${ADMIN_DIR:-${ROOT_DIR}/admin-web}"
DIST_DIR="${DIST_DIR:-${ADMIN_DIR}/dist}"
BUILD_SCRIPT="${BUILD_SCRIPT:-${ROOT_DIR}/scripts/build-admin-web.sh}"
SERVER_HOST="${SERVER_HOST:-}"
SSH_PORT="${SSH_PORT:-}"
REMOTE_ADMIN_DIR="${REMOTE_ADMIN_DIR:-/srv/caipu-miniapp/admin-web}"
REMOTE_TMP_DIR="${REMOTE_TMP_DIR:-${REMOTE_ADMIN_DIR}/.upload-tmp}"
BUILD_DIST="${BUILD_DIST:-1}"
PLAN_ONLY="${PLAN_ONLY:-0}"
DOMAIN="${DOMAIN:-}"
VERIFY_URL="${VERIFY_URL:-}"
VERIFY_HTTP="${VERIFY_HTTP:-1}"
KEEP_REMOTE_BACKUP="${KEEP_REMOTE_BACKUP:-1}"
KEEP_REMOTE_BACKUP_COUNT="${KEEP_REMOTE_BACKUP_COUNT:-3}"
SSH_BIN="${SSH_BIN:-ssh}"
SCP_BIN="${SCP_BIN:-scp}"
CURL_BIN="${CURL_BIN:-curl}"
TAR_BIN="${TAR_BIN:-tar}"
SSH_CONFIG_FILE="${SSH_CONFIG_FILE:-${HOME}/.ssh/config}"
DEFAULT_SSH_HOST_CANDIDATES="${DEFAULT_SSH_HOST_CANDIDATES:-one-hub-server oh-prod my-cloud}"

log() {
  echo "==> $*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

ssh_config_has_host_alias() {
  local target="$1"

  [[ -f "$SSH_CONFIG_FILE" ]] || return 1

  awk -v target="$target" '
    /^[[:space:]]*Host[[:space:]]+/ {
      for (i = 2; i <= NF; i++) {
        if ($i == target) {
          found = 1
        }
      }
    }
    END {
      exit(found ? 0 : 1)
    }
  ' "$SSH_CONFIG_FILE"
}

resolve_default_server_host() {
  local candidate

  for candidate in $DEFAULT_SSH_HOST_CANDIDATES; do
    if ssh_config_has_host_alias "$candidate"; then
      echo "$candidate"
      return 0
    fi
  done

  echo "my-cloud"
}

print_usage() {
  cat <<'EOF'
Usage:
  SERVER_HOST=root@your-server bash scripts/upload-admin-web-dist.sh

Environment variables:
  SERVER_HOST=one-hub-server
  SSH_PORT=22
  DOMAIN=www.example.com
  VERIFY_URL=https://www.example.com/admin/
  BUILD_DIST=1|0
  PLAN_ONLY=1
  KEEP_REMOTE_BACKUP=1|0
  KEEP_REMOTE_BACKUP_COUNT=3

Examples:
  DOMAIN=www.example.com \
    bash scripts/upload-admin-web-dist.sh

  BUILD_DIST=0 SERVER_HOST=oh-prod \
    bash scripts/upload-admin-web-dist.sh
EOF
}

if [[ "${1:-}" == "--help" ]]; then
  print_usage
  exit 0
fi

if [[ -z "$VERIFY_URL" && -n "$DOMAIN" ]]; then
  VERIFY_URL="https://${DOMAIN}/admin/"
fi

if [[ -z "$SERVER_HOST" ]]; then
  SERVER_HOST="$(resolve_default_server_host)"
fi

case "$BUILD_DIST" in
  0|1)
    ;;
  *)
    echo "unsupported BUILD_DIST: $BUILD_DIST" >&2
    exit 1
    ;;
esac

case "$PLAN_ONLY" in
  0|1)
    ;;
  *)
    echo "unsupported PLAN_ONLY: $PLAN_ONLY" >&2
    exit 1
    ;;
esac

case "$VERIFY_HTTP" in
  0|1)
    ;;
  *)
    echo "unsupported VERIFY_HTTP: $VERIFY_HTTP" >&2
    exit 1
    ;;
esac

case "$KEEP_REMOTE_BACKUP" in
  0|1)
    ;;
  *)
    echo "unsupported KEEP_REMOTE_BACKUP: $KEEP_REMOTE_BACKUP" >&2
    exit 1
    ;;
esac

if [[ ! "$KEEP_REMOTE_BACKUP_COUNT" =~ ^[0-9]+$ ]]; then
  echo "KEEP_REMOTE_BACKUP_COUNT must be an integer" >&2
  exit 1
fi

if [[ ! -d "$ADMIN_DIR" ]]; then
  echo "admin-web directory not found: $ADMIN_DIR" >&2
  exit 1
fi

require_cmd "$SSH_BIN"
require_cmd "$SCP_BIN"
require_cmd "$TAR_BIN"
if [[ "$VERIFY_HTTP" == "1" && -n "$VERIFY_URL" ]]; then
  require_cmd "$CURL_BIN"
fi

release_id="$(date +%Y%m%d-%H%M%S)"
local_dist_status="missing"
if [[ -f "${DIST_DIR}/index.html" ]]; then
  local_dist_status="present"
fi

if [[ "$PLAN_ONLY" == "1" ]]; then
  cat <<EOF

Plan summary:
- server host: ${SERVER_HOST}
- ssh config file: ${SSH_CONFIG_FILE}
- ssh port: ${SSH_PORT:-default}
- build local dist: $( [[ "$BUILD_DIST" == "1" ]] && echo yes || echo no )
- local dist status: ${local_dist_status}
- remote admin dir: ${REMOTE_ADMIN_DIR}
- remote temp dir: ${REMOTE_TMP_DIR}
- keep remote backup: $( [[ "$KEEP_REMOTE_BACKUP" == "1" ]] && echo yes || echo no )
- verify url: ${VERIFY_URL:-skip}
- release id: ${release_id}
EOF
  exit 0
fi

if [[ "$BUILD_DIST" == "1" ]]; then
  log "building admin-web dist on local machine"
  bash "$BUILD_SCRIPT"
fi

if [[ ! -f "${DIST_DIR}/index.html" ]]; then
  echo "dist build output not found: ${DIST_DIR}/index.html" >&2
  exit 1
fi

tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/caipu-admin-web-upload.XXXXXX")"
archive_path="${tmpdir}/admin-web-dist-${release_id}.tgz"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

log "packaging dist"
"$TAR_BIN" -C "$ADMIN_DIR" -czf "$archive_path" dist

ssh_cmd=("$SSH_BIN")
scp_cmd=("$SCP_BIN")
if [[ -n "$SSH_PORT" ]]; then
  ssh_cmd+=(-p "$SSH_PORT")
  scp_cmd+=(-P "$SSH_PORT")
fi

remote_archive="${REMOTE_TMP_DIR}/admin-web-dist-${release_id}.tgz"

log "preparing remote directories"
"${ssh_cmd[@]}" "$SERVER_HOST" \
  "mkdir -p '$REMOTE_ADMIN_DIR' '$REMOTE_TMP_DIR'"

log "uploading dist archive"
"${scp_cmd[@]}" "$archive_path" "${SERVER_HOST}:${remote_archive}"

log "activating uploaded dist on remote server"
"${ssh_cmd[@]}" "$SERVER_HOST" \
  "REMOTE_ADMIN_DIR='$REMOTE_ADMIN_DIR' REMOTE_TMP_DIR='$REMOTE_TMP_DIR' REMOTE_ARCHIVE='$remote_archive' RELEASE_ID='$release_id' KEEP_REMOTE_BACKUP='$KEEP_REMOTE_BACKUP' KEEP_REMOTE_BACKUP_COUNT='$KEEP_REMOTE_BACKUP_COUNT' bash -s" <<'REMOTE'
set -euo pipefail

stage_dir="${REMOTE_TMP_DIR}/stage-${RELEASE_ID}"
backup_dir="${REMOTE_ADMIN_DIR}/dist.bak-${RELEASE_ID}"

rm -rf "$stage_dir"
mkdir -p "$stage_dir"
tar -xzf "$REMOTE_ARCHIVE" -C "$stage_dir"

if [[ ! -d "${stage_dir}/dist" ]]; then
  echo "uploaded archive does not contain dist/" >&2
  exit 1
fi

if [[ "$KEEP_REMOTE_BACKUP" == "1" && -d "${REMOTE_ADMIN_DIR}/dist" ]]; then
  rm -rf "$backup_dir"
  mv "${REMOTE_ADMIN_DIR}/dist" "$backup_dir"
else
  rm -rf "${REMOTE_ADMIN_DIR}/dist"
fi

mv "${stage_dir}/dist" "${REMOTE_ADMIN_DIR}/dist"
rm -rf "$stage_dir"
rm -f "$REMOTE_ARCHIVE"

if [[ "$KEEP_REMOTE_BACKUP" == "1" ]]; then
  mapfile -t backups < <(find "$REMOTE_ADMIN_DIR" -maxdepth 1 -mindepth 1 -type d -name 'dist.bak-*' | sort -r)
  if (( ${#backups[@]} > KEEP_REMOTE_BACKUP_COUNT )); then
    for ((i=KEEP_REMOTE_BACKUP_COUNT; i<${#backups[@]}; i++)); do
      rm -rf "${backups[$i]}"
    done
  fi
else
  find "$REMOTE_ADMIN_DIR" -maxdepth 1 -mindepth 1 -type d -name 'dist.bak-*' -exec rm -rf {} +
fi

test -f "${REMOTE_ADMIN_DIR}/dist/index.html"
REMOTE

if [[ "$VERIFY_HTTP" == "1" && -n "$VERIFY_URL" ]]; then
  log "verifying public admin url"
  "$CURL_BIN" -I --max-time 15 "$VERIFY_URL"
fi

cat <<EOF

Upload completed.

Summary:
- server host: ${SERVER_HOST}
- remote dist dir: ${REMOTE_ADMIN_DIR}/dist
- release id: ${release_id}
- verify url: ${VERIFY_URL:-skip}
EOF
