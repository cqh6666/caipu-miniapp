#!/usr/bin/env bash
set -euo pipefail

# Safety tombstone: keep this path so unknown cron/CI aliases fail closed instead
# of silently falling back to the former mutable-binary deployment flow.
cat >&2 <<'EOF'
backend/scripts/deploy.sh 是刻意保留的安全 tombstone，旧覆盖式发布已永久禁用。

- 已登录服务器：bash scripts/deploy-backend-on-server.sh
- 从本地通过 SSH：bash backend/scripts/deploy-server-build.sh

新入口会执行配置校验、一致性备份、迁移预检、版本化原子切换、
连续 readiness 检查，并在失败时恢复上一二进制。

在完成服务器 cron、外部 CI 和个人命令别名审计前，不得删除此路径，
也不得在这里恢复任何实际发布逻辑。
EOF
exit 2
