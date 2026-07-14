export function createActionFeedbackController(options = {}) {
	const {
		onState = () => {},
		minDuration = 1200,
		setTimeoutFn = setTimeout,
		clearTimeoutFn = clearTimeout
	} = options
	let timer = null
	let tick = 0

	function clearTimer() {
		if (timer !== null) clearTimeoutFn(timer)
		timer = null
	}

	return {
		show(value = {}) {
			const title = String(value?.title || '').trim()
			if (!title) return false
			clearTimer()
			tick += 1
			onState({
				visible: true,
				tone: String(value?.tone || 'done').trim() || 'done',
				title,
				description: String(value?.description || '').trim(),
				showSparkles: !!value?.showSparkles,
				tick
			})
			timer = setTimeoutFn(() => {
				timer = null
				onState({ visible: false, tick })
			}, Math.max(minDuration, Number(value?.duration) || 1680))
			return true
		},
		clear() {
			clearTimer()
			onState({ visible: false, tone: '', title: '', description: '', showSparkles: false, tick })
		},
		status() {
			return { active: timer !== null, tick }
		}
	}
}
