export function hasParseableShareHint(text = '') {
	const value = String(text || '').trim()
	if (!value) return false
	if (/https?:\/\//i.test(value)) return true
	return /小红书|b\s*站|bilibili|b23\.tv|xhslink|xiaohongshu|抖音|快手|美团|大众点评|点评|douyin|kuaishou|dpurl/i.test(value)
}

export function readClipboardText(api, onError = null) {
	return new Promise((resolve) => {
		if (!api || typeof api.getClipboardData !== 'function') {
			resolve('')
			return
		}
		api.getClipboardData({
			success: (result) => resolve(String(result?.data || '').trim()),
			fail: (error) => {
				if (typeof onError === 'function') onError(error)
				resolve('')
			}
		})
	})
}

export function delay(ms) {
	return new Promise((resolve) => setTimeout(resolve, ms))
}

export function createAddPreviewFlowController(options = {}) {
	const {
		onState = () => {},
		interval = 1000,
		setIntervalFn = setInterval,
		clearIntervalFn = clearInterval
	} = options
	let timer = null
	let sequence = 0
	let activeRunId = 0
	let duration = 0
	let stage = 'extracting'

	function publish(isParsing) {
		onState({ isParsing, stage, duration })
	}

	function stop(runId = activeRunId) {
		if (!activeRunId || runId !== activeRunId) return false
		if (timer) clearIntervalFn(timer)
		timer = null
		activeRunId = 0
		duration = 0
		stage = 'extracting'
		publish(false)
		return true
	}

	return {
		start(initialStage = 'extracting') {
			if (activeRunId) stop(activeRunId)
			activeRunId = ++sequence
			duration = 0
			stage = initialStage
			publish(true)
			timer = setIntervalFn(() => {
				duration += 1
				publish(true)
			}, interval)
			return activeRunId
		},
		setStage(runId, nextStage) {
			if (!activeRunId || runId !== activeRunId) return false
			stage = String(nextStage || 'extracting')
			publish(true)
			return true
		},
		stop,
		isCurrent(runId) {
			return !!activeRunId && runId === activeRunId
		},
		status() {
			return { active: !!activeRunId, runId: activeRunId, stage, duration }
		}
	}
}
