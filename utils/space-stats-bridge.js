// 首页 ↔ 空间洞察页 的轻量数据桥：
// - context：进入洞察页时携带的统计快照 + kitchenId（避免走 query 传大对象）。
// - pendingAction：洞察页里点击的动作（进店详情 / 看草稿等），由首页 onShow 消费并执行页内切换。
// 纯内存单例，不做持久化；小程序场景下页面栈存活期间足够。

let context = null
let pendingAction = null

export function setSpaceStatsContext(value) {
	context = value || null
}

export function takeSpaceStatsContext() {
	const value = context
	context = null
	return value
}

export function setPendingSpaceStatsAction(action) {
	pendingAction = action || null
}

export function takePendingSpaceStatsAction() {
	const value = pendingAction
	pendingAction = null
	return value
}
