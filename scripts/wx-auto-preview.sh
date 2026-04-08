#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/unpackage/dist/dev/mp-weixin"

SKIP_COMPILE=0
DEBUG_MODE=0
CLI_LANG="${WX_CLI_LANG:-zh}"
CLI_PORT="${WX_CLI_PORT:-}"

print_help() {
	cat <<'EOF'
用法：
  bash scripts/wx-auto-preview.sh [--skip-compile] [--debug] [--port <port>]

说明：
  1. 可选先调用 HBuilderX CLI 编译当前项目到微信小程序产物目录
  2. 自动打开微信开发者工具中的产物项目
  3. 触发微信开发者工具 CLI 的 auto-preview

参数：
  --skip-compile   跳过 HBuilderX 编译，直接对现有产物触发自动预览
  --debug          为微信开发者工具 CLI 打开 debug 输出
  --port <port>    显式指定微信开发者工具 HTTP 服务端口
  -h, --help       查看帮助

环境变量：
  HBUILDERX_CLI         自定义 HBuilderX CLI 路径
  WECHAT_DEVTOOLS_CLI   自定义微信开发者工具 CLI 路径
  WX_CLI_LANG           微信 CLI 语言，默认 zh
  WX_CLI_PORT           微信 CLI 端口，可替代 --port
EOF
}

log() {
	printf '[wx-auto-preview] %s\n' "$*"
}

fail() {
	printf '[wx-auto-preview] 错误：%s\n' "$*" >&2
	exit 1
}

resolve_cli_from_candidates() {
	local override="$1"
	local bundle_id="$2"
	shift 2

	if [[ -n "${override}" ]]; then
		[[ -x "${override}" ]] || fail "指定的 CLI 不可执行：${override}"
		printf '%s\n' "${override}"
		return 0
	fi

	local candidate
	for candidate in "$@"; do
		if [[ -x "${candidate}" ]]; then
			printf '%s\n' "${candidate}"
			return 0
		fi
	done

	if command -v mdfind >/dev/null 2>&1; then
		local app_path
		app_path="$(mdfind "kMDItemCFBundleIdentifier == '${bundle_id}'" | head -n 1 || true)"
		if [[ -n "${app_path}" && -x "${app_path}/Contents/MacOS/cli" ]]; then
			printf '%s\n' "${app_path}/Contents/MacOS/cli"
			return 0
		fi
	fi

	return 1
}

resolve_hbuilderx_cli() {
	resolve_cli_from_candidates \
		"${HBUILDERX_CLI:-}" \
		"io.dcloud.HBuilderX" \
		"/Applications/HBuilderX.app/Contents/MacOS/cli" \
		"${HOME}/Applications/HBuilderX.app/Contents/MacOS/cli"
}

resolve_wechat_cli() {
	resolve_cli_from_candidates \
		"${WECHAT_DEVTOOLS_CLI:-}" \
		"com.tencent.webplusdevtools" \
		"/Applications/wechatwebdevtools.app/Contents/MacOS/cli" \
		"/Applications/微信开发者工具.app/Contents/MacOS/cli" \
		"${HOME}/Applications/wechatwebdevtools.app/Contents/MacOS/cli" \
		"${HOME}/Applications/微信开发者工具.app/Contents/MacOS/cli"
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--skip-compile)
			SKIP_COMPILE=1
			shift
			;;
		--debug)
			DEBUG_MODE=1
			shift
			;;
		--port)
			[[ $# -ge 2 ]] || fail "--port 需要一个端口值"
			CLI_PORT="$2"
			shift 2
			;;
		-h|--help)
			print_help
			exit 0
			;;
		*)
			fail "未知参数：$1。可使用 --help 查看帮助。"
			;;
	esac
done

[[ "$(uname -s)" == "Darwin" ]] || fail "当前脚本仅支持 macOS。"

WECHAT_CLI="$(resolve_wechat_cli)" || fail "未找到微信开发者工具 CLI，请安装后重试，或通过 WECHAT_DEVTOOLS_CLI 指定路径。"
HBUILDERX_CLI=""

if [[ "${SKIP_COMPILE}" -eq 0 ]]; then
	HBUILDERX_CLI="$(resolve_hbuilderx_cli)" || fail "未找到 HBuilderX CLI，请安装后重试，或通过 HBUILDERX_CLI 指定路径。"
fi

WECHAT_ARGS=(--project "${DIST_DIR}" --lang "${CLI_LANG}")

if [[ -n "${CLI_PORT}" ]]; then
	WECHAT_ARGS+=(--port "${CLI_PORT}")
fi

if [[ "${DEBUG_MODE}" -eq 1 ]]; then
	WECHAT_ARGS+=(--debug)
fi

if [[ "${SKIP_COMPILE}" -eq 0 ]]; then
	log "使用 HBuilderX CLI 编译微信小程序产物"
	"${HBUILDERX_CLI}" launch mp-weixin --project "${PROJECT_ROOT}" --compile true
else
	log "跳过编译，直接使用现有微信小程序产物"
fi

[[ -f "${DIST_DIR}/project.config.json" ]] || fail "未找到微信小程序产物：${DIST_DIR}/project.config.json"

log "打开微信开发者工具项目：${DIST_DIR}"
"${WECHAT_CLI}" open "${WECHAT_ARGS[@]}"

log "触发自动预览"
"${WECHAT_CLI}" auto-preview "${WECHAT_ARGS[@]}"

log "自动预览命令已触发完成"
