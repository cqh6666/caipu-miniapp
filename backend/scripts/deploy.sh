#!/usr/bin/env bash
set -euo pipefail

cat >&2 <<'EOF'
backend/scripts/deploy.sh 已废弃，避免继续使用覆盖式发布。

- 已登录服务器：bash scripts/deploy-backend-on-server.sh
- 从本地通过 SSH：bash backend/scripts/deploy-server-build.sh

新入口会执行配置校验、一致性备份、迁移预检、版本化原子切换、
连续 readiness 检查，并在失败时恢复上一二进制。
EOF
exit 2
