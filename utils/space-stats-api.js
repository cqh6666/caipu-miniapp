// 空间统计能力设计 §9.5：后端统计接口封装。
// GET /api/kitchens/{kitchenID}/stats?window=7d|30d|90d|all
// 前端 V2 优先用后端 stats（趋势 / 成员贡献 / 消费统计），失败时由页面回退本地聚合。

import { request } from './http'
import { mapRemoteStatsToViewModel } from './space-stats'

const VALID_WINDOWS = ['7d', '30d', '90d', 'all']

export function normalizeStatsWindow(window) {
	const value = String(window || '').trim().toLowerCase()
	return VALID_WINDOWS.includes(value) ? value : '30d'
}

// 返回归一后的视图模型（与 buildSpaceStats 同构），失败时抛出错误交页面处理。
export function getKitchenStats(kitchenId, options = {}) {
	const id = Number(kitchenId) || 0
	if (!id) {
		return Promise.reject(new Error('缺少空间 ID'))
	}

	const window = normalizeStatsWindow(options.window)
	return request({
		url: `/caipu-api/kitchens/${id}/stats`,
		method: 'GET',
		data: { window }
	}).then((data) => {
		const remote = data?.stats && typeof data.stats === 'object' ? data.stats : {}
		return mapRemoteStatsToViewModel(remote, { window, isSyncing: false })
	})
}
