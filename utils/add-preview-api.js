import { request } from './http'

export function previewAddLink(kitchenId, payload = {}) {
	return request({
		url: `/caipu-api/kitchens/${kitchenId}/add-link-previews`,
		method: 'POST',
		data: {
			text: payload.text || '',
			city: payload.city || '',
			latitude: Number(payload.latitude) || 0,
			longitude: Number(payload.longitude) || 0,
			limit: Number(payload.limit) || 3
		},
		// 小红书/B站解析涉及 sidecar 抓取 + AI 总结，实测约 17-30s。这里刻意把前端超时收在
		// 22s（低于后端 30s）：解析过慢时主动放弃实时草稿，交由「超时转后台」兜底——前端用
		// 链接+猜测标题先建占位菜谱，后端 parse_status=pending 队列稍后自动补全。
		// 详见 docs/add-recipe-async-timeout-design.md。
		timeout: 22000
	}).then((data) => data?.result || null)
}

// isPreviewTimeoutError：仅在「明确是请求超时」时返回 true，用于决定是否走超时转后台兜底。
// uni.request 超时 fail 的 errMsg 形如 "request:fail timeout"，经 http.js 归一化为 error.message。
export function isPreviewTimeoutError(error) {
	const message = String(error?.message || error?.errMsg || '').toLowerCase()
	return message.includes('timeout') || message.includes('超时')
}
