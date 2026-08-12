#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
source "$ROOT_DIR/scripts/lib/host-resources.sh"
REPO_DIR="${REPO_DIR:-$ROOT_DIR}"
SIDECAR_DIR="${SIDECAR_DIR:-${REPO_DIR}/sidecars/linkparse-sidecar}"
SERVICE_NAME="${SERVICE_NAME:-caipu-linkparse-sidecar}"
SYSTEMCTL_BIN="${SYSTEMCTL_BIN:-systemctl}"
RUN_GIT_PULL="${RUN_GIT_PULL:-1}"
PLAN_ONLY="${PLAN_ONLY:-0}"
BUILD_NICE="${BUILD_NICE:-10}"
NPM_CACHE_DIR="${NPM_CACHE_DIR:-/tmp/caipu-sidecar-npm-cache}"
SIDECAR_INSTALL_MODE="${SIDECAR_INSTALL_MODE:-auto}"
ALLOW_LOW_RESOURCE_BUILD="${ALLOW_LOW_RESOURCE_BUILD:-0}"
LOW_RESOURCE_MIN_CPU="${LOW_RESOURCE_MIN_CPU:-4}"
LOW_RESOURCE_MIN_MEM_MB="${LOW_RESOURCE_MIN_MEM_MB:-3072}"
LOW_RESOURCE_MIN_SWAP_MB="${LOW_RESOURCE_MIN_SWAP_MB:-1024}"
HEALTHCHECK_URL="${HEALTHCHECK_URL:-http://127.0.0.1:8091/v1/health}"

log() {
  echo "==> $*"
}

print_usage() {
  cat <<'EOF'
Usage:
  bash scripts/deploy-linkparse-sidecar-on-server.sh

Environment variables:
  RUN_GIT_PULL=1|0
  PLAN_ONLY=1
  SIDECAR_INSTALL_MODE=auto|always|never
  ALLOW_LOW_RESOURCE_BUILD=1
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

case "$SIDECAR_INSTALL_MODE" in
  auto|always|never)
    ;;
  *)
    echo "unsupported SIDECAR_INSTALL_MODE: $SIDECAR_INSTALL_MODE" >&2
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

restart_sidecar=0
install_sidecar_deps=0

if [[ -n "$changed_files" ]]; then
  while IFS= read -r file; do
    case "$file" in
      sidecars/linkparse-sidecar/*)
        restart_sidecar=1
        ;;
    esac

    case "$file" in
      sidecars/linkparse-sidecar/package.json|sidecars/linkparse-sidecar/package-lock.json)
        install_sidecar_deps=1
        ;;
    esac
  done <<< "$changed_files"
fi

if [[ "$SIDECAR_INSTALL_MODE" == "always" ]]; then
  install_sidecar_deps=1
  restart_sidecar=1
fi

if [[ "$SIDECAR_INSTALL_MODE" == "never" ]]; then
  install_sidecar_deps=0
fi

if [[ "$install_sidecar_deps" == "0" && ! -d "$SIDECAR_DIR/node_modules" ]]; then
  install_sidecar_deps=1
  restart_sidecar=1
fi

if [[ "$PLAN_ONLY" == "1" ]]; then
  cat <<EOF

Plan summary:
- git range: ${before_commit} -> ${after_commit}
- install sidecar deps: $( [[ "$install_sidecar_deps" == "1" ]] && echo yes || echo no )
- restart sidecar: $( [[ "$restart_sidecar" == "1" ]] && echo yes || echo no )

$(host_resources_print_summary)
EOF
  exit 0
fi

if [[ "$install_sidecar_deps" == "0" && "$restart_sidecar" == "0" ]]; then
  if [[ "$before_commit" == "$after_commit" ]]; then
    log "repository already up to date, nothing to deploy"
  else
    log "no linkparse-sidecar changes detected, skipping install and restart"
  fi
  exit 0
fi

if host_resources_is_low_resource "$LOW_RESOURCE_MIN_CPU" "$LOW_RESOURCE_MIN_MEM_MB" "$LOW_RESOURCE_MIN_SWAP_MB" && [[ "$install_sidecar_deps" == "1" ]] && [[ "$ALLOW_LOW_RESOURCE_BUILD" != "1" ]]; then
  cat <<EOF
Refusing to run sidecar npm install on this low-resource host.

$(host_resources_print_summary)

Requested work:
- install sidecar deps: $( [[ "$install_sidecar_deps" == "1" ]] && echo yes || echo no )
- restart sidecar: $( [[ "$restart_sidecar" == "1" ]] && echo yes || echo no )

Suggestions:
- Preview the plan safely: PLAN_ONLY=1 bash scripts/deploy-linkparse-sidecar-on-server.sh
- Pull source only: RUN_GIT_PULL=1 PLAN_ONLY=1 bash scripts/deploy-linkparse-sidecar-on-server.sh
- If only JS files changed and dependencies did not change, keep current node_modules and rerun with SIDECAR_INSTALL_MODE=never
- If you still need to force npm install in a maintenance window, rerun with ALLOW_LOW_RESOURCE_BUILD=1
EOF
  exit 2
fi

cd "$SIDECAR_DIR"

if [[ "$install_sidecar_deps" == "1" ]]; then
  log "installing sidecar dependencies"
  host_resources_run_low_priority "$BUILD_NICE" env \
    npm_config_cache="$NPM_CACHE_DIR" \
    npm_config_audit=false \
    npm_config_fund=false \
    npm install --prefer-offline
fi

if [[ "$restart_sidecar" == "1" || "$install_sidecar_deps" == "1" ]]; then
  log "restarting sidecar service"
  "$SYSTEMCTL_BIN" restart "$SERVICE_NAME"
  "$SYSTEMCTL_BIN" status "$SERVICE_NAME" --no-pager

  log "checking sidecar health"
  curl --fail --silent "$HEALTHCHECK_URL" >/dev/null
fi

cat <<EOF

Deploy completed.

Summary:
- git range: ${before_commit} -> ${after_commit}
- install sidecar deps: $( [[ "$install_sidecar_deps" == "1" ]] && echo yes || echo no )
- restart sidecar: $( [[ "$restart_sidecar" == "1" || "$install_sidecar_deps" == "1" ]] && echo yes || echo no )
EOF
