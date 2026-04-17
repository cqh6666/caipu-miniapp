#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_DIR="${REPO_DIR:-$ROOT_DIR}"
BACKEND_DIR="${BACKEND_DIR:-${REPO_DIR}/backend}"
ADMIN_WEB_DIR="${ADMIN_WEB_DIR:-${REPO_DIR}/admin-web}"
BINARY_PATH="${BINARY_PATH:-${BACKEND_DIR}/bin/server}"
SERVICE_NAME="${SERVICE_NAME:-caipu-backend}"
SYSTEMCTL_BIN="${SYSTEMCTL_BIN:-systemctl}"
APP_PORT="${APP_PORT:-8080}"
HEALTHCHECK_PATH="${HEALTHCHECK_PATH:-/healthz}"
DEPLOY_SCOPE="${DEPLOY_SCOPE:-auto}"
RUN_GIT_PULL="${RUN_GIT_PULL:-1}"
BUILD_NICE="${BUILD_NICE:-10}"
GO_BUILD_GOMAXPROCS="${GO_BUILD_GOMAXPROCS:-1}"
GOCACHE_DIR="${GOCACHE_DIR:-/tmp/caipu-go-build-cache}"
NPM_CACHE_DIR="${NPM_CACHE_DIR:-/tmp/caipu-npm-cache}"
ADMIN_WEB_INSTALL_MODE="${ADMIN_WEB_INSTALL_MODE:-auto}"
ADMIN_WEB_NODE_OPTIONS="${ADMIN_WEB_NODE_OPTIONS:---max-old-space-size=768}"
PLAN_ONLY="${PLAN_ONLY:-0}"
ALLOW_LOW_RESOURCE_BUILD="${ALLOW_LOW_RESOURCE_BUILD:-0}"
LOW_RESOURCE_MIN_CPU="${LOW_RESOURCE_MIN_CPU:-4}"
LOW_RESOURCE_MIN_MEM_MB="${LOW_RESOURCE_MIN_MEM_MB:-3072}"
LOW_RESOURCE_MIN_SWAP_MB="${LOW_RESOURCE_MIN_SWAP_MB:-1024}"
GO_BIN_DIR="${GO_BIN_DIR:-/usr/local/go/bin}"

log() {
  echo "==> $*"
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

ensure_go_in_path() {
  if has_cmd go; then
    return 0
  fi

  if [[ -x "${GO_BIN_DIR}/go" ]]; then
    export PATH="${GO_BIN_DIR}:$PATH"
    return 0
  fi

  echo "go not found in PATH and fallback path is missing: ${GO_BIN_DIR}/go" >&2
  return 1
}

run_low_priority() {
  if has_cmd ionice; then
    ionice -c3 nice -n "$BUILD_NICE" "$@"
    return
  fi

  nice -n "$BUILD_NICE" "$@"
}

read_cpu_count() {
  if has_cmd nproc; then
    nproc
    return
  fi

  echo 1
}

read_mem_total_mb() {
  awk '/MemTotal:/ { printf "%d\n", $2 / 1024 }' /proc/meminfo
}

read_swap_total_mb() {
  awk '/SwapTotal:/ { printf "%d\n", $2 / 1024 }' /proc/meminfo
}

is_low_resource_host() {
  local cpu_count mem_total_mb swap_total_mb

  cpu_count="$(read_cpu_count)"
  mem_total_mb="$(read_mem_total_mb)"
  swap_total_mb="$(read_swap_total_mb)"

  if (( cpu_count < LOW_RESOURCE_MIN_CPU )); then
    return 0
  fi

  if (( mem_total_mb < LOW_RESOURCE_MIN_MEM_MB )); then
    return 0
  fi

  if (( swap_total_mb < LOW_RESOURCE_MIN_SWAP_MB )); then
    return 0
  fi

  return 1
}

requires_heavy_frontend_build() {
  [[ "$build_admin_web" == "1" ]]
}

print_resource_summary() {
  local cpu_count mem_total_mb swap_total_mb

  cpu_count="$(read_cpu_count)"
  mem_total_mb="$(read_mem_total_mb)"
  swap_total_mb="$(read_swap_total_mb)"

  cat <<EOF
Host resources:
- cpu: ${cpu_count} vCPU
- memory: ${mem_total_mb} MiB
- swap: ${swap_total_mb} MiB
EOF
}

print_usage() {
  cat <<'EOF'
Usage:
  bash scripts/deploy-on-server.sh

Environment variables:
  DEPLOY_SCOPE=auto|all|backend|admin-web|none
  RUN_GIT_PULL=1|0
  BUILD_NICE=10
  GO_BUILD_GOMAXPROCS=1
  ADMIN_WEB_INSTALL_MODE=auto|always|never
  ADMIN_WEB_NODE_OPTIONS=--max-old-space-size=768
  PLAN_ONLY=1
  ALLOW_LOW_RESOURCE_BUILD=1

Recommended wrappers:
  bash scripts/deploy-backend-on-server.sh
  bash scripts/deploy-admin-web-on-server.sh
  bash scripts/deploy-linkparse-sidecar-on-server.sh
EOF
}

if [[ "${1:-}" == "--help" ]]; then
  print_usage
  exit 0
fi

if [[ ! -d "$REPO_DIR/.git" ]]; then
  echo "repository directory not found: $REPO_DIR" >&2
  exit 1
fi

case "$DEPLOY_SCOPE" in
  auto|all|backend|admin-web|none)
    ;;
  *)
    echo "unsupported DEPLOY_SCOPE: $DEPLOY_SCOPE" >&2
    exit 1
    ;;
esac

case "$ADMIN_WEB_INSTALL_MODE" in
  auto|always|never)
    ;;
  *)
    echo "unsupported ADMIN_WEB_INSTALL_MODE: $ADMIN_WEB_INSTALL_MODE" >&2
    exit 1
    ;;
esac

cd "$REPO_DIR"

before_commit="$(git rev-parse HEAD)"
after_commit="$before_commit"

if [[ "$RUN_GIT_PULL" == "1" ]]; then
  log "updating repository"
  git pull --ff-only
  after_commit="$(git rev-parse HEAD)"
fi

changed_files=""
if [[ "$before_commit" != "$after_commit" ]]; then
  changed_files="$(git diff --name-only "$before_commit" "$after_commit")"
fi

build_backend=0
build_admin_web=0
restart_backend=0
install_admin_deps=0

if [[ "$DEPLOY_SCOPE" == "all" ]]; then
  build_backend=1
  build_admin_web=1
fi

if [[ "$DEPLOY_SCOPE" == "backend" ]]; then
  build_backend=1
fi

if [[ "$DEPLOY_SCOPE" == "admin-web" ]]; then
  build_admin_web=1
fi

if [[ "$DEPLOY_SCOPE" == "auto" && -n "$changed_files" ]]; then
  while IFS= read -r file; do
    case "$file" in
      backend/*)
        build_backend=1
        ;;
      admin-web/*|scripts/build-admin-web.sh)
        build_admin_web=1
        ;;
    esac

    case "$file" in
      admin-web/package.json|admin-web/package-lock.json)
        install_admin_deps=1
        ;;
    esac
  done <<< "$changed_files"
fi

if [[ "$ADMIN_WEB_INSTALL_MODE" == "always" ]]; then
  install_admin_deps=1
fi

if [[ "$ADMIN_WEB_INSTALL_MODE" == "never" ]]; then
  install_admin_deps=0
fi

if [[ "$build_admin_web" == "1" && "$install_admin_deps" == "0" && ! -d "$ADMIN_WEB_DIR/node_modules" ]]; then
  install_admin_deps=1
fi

if [[ "$DEPLOY_SCOPE" == "none" ]]; then
  log "DEPLOY_SCOPE=none, skipping build and restart"
  exit 0
fi

if [[ "$PLAN_ONLY" == "1" ]]; then
  cat <<EOF

Plan summary:
- git range: ${before_commit} -> ${after_commit}
- build backend: $( [[ "$build_backend" == "1" ]] && echo yes || echo no )
- build admin-web: $( [[ "$build_admin_web" == "1" ]] && echo yes || echo no )
- restart backend: $( [[ "$build_backend" == "1" ]] && echo yes || echo no )

$(print_resource_summary)
EOF
  exit 0
fi

if [[ "$build_backend" == "0" && "$build_admin_web" == "0" ]]; then
  if [[ "$before_commit" == "$after_commit" ]]; then
    log "repository already up to date, nothing to deploy"
  else
    log "no backend/admin-web changes detected, skipping build and restart"
  fi
  exit 0
fi

if is_low_resource_host && requires_heavy_frontend_build && [[ "$ALLOW_LOW_RESOURCE_BUILD" != "1" ]]; then
  cat <<EOF
Refusing to run admin-web build on this low-resource host.

$(print_resource_summary)

Requested work:
- build backend: $( [[ "$build_backend" == "1" ]] && echo yes || echo no )
- build admin-web: $( [[ "$build_admin_web" == "1" ]] && echo yes || echo no )

Suggestions:
- Preview the plan safely: PLAN_ONLY=1 bash scripts/deploy-admin-web-on-server.sh
- Pull source only: DEPLOY_SCOPE=none bash scripts/deploy-on-server.sh
- Backend-only deploy can still run on this host:
  bash scripts/deploy-backend-on-server.sh
- Prefer building admin-web on another machine or CI, then upload artifacts to this server
- If you still need to force an admin-web build in a maintenance window, rerun with ALLOW_LOW_RESOURCE_BUILD=1
EOF
  exit 2
fi

if is_low_resource_host && [[ "$build_backend" == "1" ]] && [[ "$build_admin_web" == "0" ]]; then
  log "low-resource host detected; allowing backend-only build, but admin-web build remains blocked by default"
fi

if [[ "$build_backend" == "1" ]]; then
  log "building backend binary with low priority"
  cd "$BACKEND_DIR"
  mkdir -p "$(dirname "$BINARY_PATH")"
  ensure_go_in_path
  run_low_priority env \
    GOMAXPROCS="$GO_BUILD_GOMAXPROCS" \
    GOCACHE="$GOCACHE_DIR" \
    go build -o "$BINARY_PATH" ./cmd/server
  restart_backend=1
fi

if [[ "$build_admin_web" == "1" ]]; then
  log "building admin-web with low priority"
  cd "$ADMIN_WEB_DIR"

  if [[ "$install_admin_deps" == "1" ]]; then
    log "installing admin-web dependencies"
    run_low_priority env \
      npm_config_cache="$NPM_CACHE_DIR" \
      npm_config_audit=false \
      npm_config_fund=false \
      npm install --prefer-offline
  else
    log "skipping npm install because lockfile did not change"
  fi

  run_low_priority env \
    NODE_OPTIONS="$ADMIN_WEB_NODE_OPTIONS" \
    npm_config_cache="$NPM_CACHE_DIR" \
    npm run build
fi

if [[ "$restart_backend" == "1" ]]; then
  log "restarting backend service"
  "$SYSTEMCTL_BIN" restart "$SERVICE_NAME"
  "$SYSTEMCTL_BIN" status "$SERVICE_NAME" --no-pager

  log "checking backend health"
  curl --fail --silent "http://127.0.0.1:${APP_PORT}${HEALTHCHECK_PATH}" >/dev/null
else
  log "backend code unchanged, skip service restart"
fi

cat <<EOF

Deploy completed.

Summary:
- git range: ${before_commit} -> ${after_commit}
- build backend: $( [[ "$build_backend" == "1" ]] && echo yes || echo no )
- build admin-web: $( [[ "$build_admin_web" == "1" ]] && echo yes || echo no )
- restart backend: $( [[ "$restart_backend" == "1" ]] && echo yes || echo no )

Suggestions:
- Need both backend and admin-web: DEPLOY_SCOPE=all bash scripts/deploy-on-server.sh
- Backend only: bash scripts/deploy-backend-on-server.sh
- Admin only: bash scripts/deploy-admin-web-on-server.sh
- Sidecar only: bash scripts/deploy-linkparse-sidecar-on-server.sh
- If package lock unchanged but build still fails, rerun with ADMIN_WEB_INSTALL_MODE=always
EOF
