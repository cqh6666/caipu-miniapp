# 后端运维告警策略

- **最后更新**：2026-07-16 11:10:45 +0800
- **适用范围**：单机 systemd + SQLite 的 `caipu-backend`
- **仓库状态**：巡检脚本、systemd timer 和日志留存模板已完成；生产告警接收端待接入

## 1. 目标与边界

仓库通过 `backend/scripts/check-operational-alerts.sh` 将 5xx、worker error、磁盘、备份
年龄和恢复演练年龄收敛为可自动执行的检查。脚本输出 `OK/WARNING/CRITICAL`，退出码分别
为 `0/1/2`；bootstrap 每五分钟运行 `${SERVICE_NAME}-ops-health.timer`。

这套实现负责产生可靠的本机告警信号，不假装已经完成短信、邮件或集中监控投递。生产需让
现有监控平台订阅 `caipu-backend-ops-health.service` 的非零状态或 journal 输出，并完成一次
真实告警投递验收。

## 2. 默认阈值

| 信号 | Warning | Critical | 默认窗口/来源 |
| --- | ---: | ---: | --- |
| HTTP 5xx | 1 次 | 5 次 | 最近 5 分钟 `request completed status=5xx` |
| worker/job error | 1 次 | 3 次 | 最近 5 分钟 Error 日志 |
| 磁盘使用率 | 80% | 90% | `DISK_PATH` 所在文件系统 |
| 最新备份年龄 | — | 26 小时 | `BACKUP_ROOT/backup-*` mtime |
| 最近恢复校验成功年龄 | — | 8 天 | `.last-restore-drill-ok` |

阈值可通过同名环境变量覆盖：

```text
HTTP_5XX_WARNING_COUNT / HTTP_5XX_CRITICAL_COUNT
WORKER_ERROR_WARNING_COUNT / WORKER_ERROR_CRITICAL_COUNT
DISK_WARNING_PERCENT / DISK_CRITICAL_PERCENT
BACKUP_MAX_AGE_SECONDS / RESTORE_DRILL_MAX_AGE_SECONDS
JOURNAL_SINCE
```

修改生产阈值前应先核对真实请求量和基线噪声，并把原因写入正式变更记录。

## 3. 日志留存与查询

bootstrap 写入 `/etc/systemd/journald.conf.d/caipu-backend.conf`：

```ini
[Journal]
SystemMaxUse=512M
MaxRetentionSec=14day
Compress=yes
```

`SystemMaxUse` 是主机全局 journald 上限，不是单服务配额。共享主机应结合其他服务日志量
调整 `JOURNAL_SYSTEM_MAX_USE` / `JOURNAL_MAX_RETENTION_SEC`，或接入集中采集后缩短本机
保留期。后端 unit 同时设置 30 秒 1000 条的日志速率上限。

常用查询：

```bash
journalctl -u caipu-backend --since '-15 minutes' --no-pager
journalctl -u caipu-backend --since today | rg 'request_id=<request-id>'
journalctl -u caipu-backend --since today | rg 'release_id=<release-id>'
systemctl status caipu-backend-ops-health.service --no-pager
journalctl -u caipu-backend-ops-health.service -n 100 --no-pager
```

## 4. 备份与恢复信号

`verify-latest-backup.sh` 只有在 SHA-256、SQLite quick check 和 uploads 解包全部通过后，
才会原子更新 `BACKUP_ROOT/.last-restore-drill-ok`。巡检只读取该成功标记，失败演练不会刷新
时间，从而在 8 天后持续产生 Critical。

备份年龄只证明最近生成过本机包，不证明异机副本可用。生产仍需：

1. 配置 `OFFSITE_BACKUP_TARGET`，建议启用 `REQUIRE_OFFSITE_BACKUP=1`。
2. 至少完成一次从异机副本恢复 SQLite 和 uploads 的实机演练。
3. 记录恢复耗时、校验结果和可接受的数据丢失窗口。

## 5. 响应流程

1. 先记录告警首次时间、主机、release ID 和最近 request ID。
2. 5xx/worker 告警先按 request ID 查看脱敏错误链，再判断是上游、数据库、配置还是版本问题。
3. 磁盘告警优先检查 journal、release、备份和 uploads，禁止未确认内容就直接删除数据库文件。
4. 备份/恢复告警先手工运行对应 verify 脚本；校验失败的包不得覆盖最近一份已知可恢复副本。
5. 若新 release 触发集中 5xx 或 readiness 失败，按版本化发布流程恢复上一二进制；migration
   只允许前向兼容，不自动反向 SQL。

## 6. 生产关闭条件

以下证据齐全后，OPS-003 的外部验收项才可关闭：

- `systemctl list-timers` 可见 backup、restore-drill、ops-health 三个 timer。
- 主动制造测试信号后，外部接收端收到 Warning/Critical，且包含主机、service、release。
- journal 能按 request ID 和 release ID 定位同一请求/构建。
- 最新备份和恢复成功标记年龄在阈值内；异机副本完成真实恢复。
- journald 实际磁盘占用和 14 天保留口径符合共享主机容量预算。
