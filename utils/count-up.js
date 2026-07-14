export function easeOutCubic(progress) {
	const normalized = Math.max(0, Math.min(Number(progress) || 0, 1))
	return 1 - Math.pow(1 - normalized, 3)
}

export function createCountUpController(options = {}) {
	const {
		read = () => 0,
		write = () => {},
		writeBatch = (values) => Object.entries(values).forEach(([key, value]) => write(key, value)),
		duration = 640,
		step = 40,
		setIntervalFn = setInterval,
		clearIntervalFn = clearInterval
	} = options
	const tasks = new Map()
	let timer = null

	function stopTimerIfIdle() {
		if (tasks.size || timer === null) return
		clearIntervalFn(timer)
		timer = null
	}

	function clear(keys = [...tasks.keys()]) {
		keys.forEach((key) => {
			tasks.delete(key)
		})
		stopTimerIfIdle()
	}

	function tick() {
		const values = {}
		tasks.forEach((task, key) => {
			task.currentStep += 1
			const progress = Math.min(1, task.currentStep / task.totalSteps)
			const nextValue = task.start + (task.target - task.start) * easeOutCubic(progress)
			values[key] = task.round ? Math.round(nextValue) : nextValue
			if (progress >= 1) {
				values[key] = task.target
				tasks.delete(key)
			}
		})
		if (Object.keys(values).length) writeBatch(values)
		stopTimerIfIdle()
	}

	function ensureTimer() {
		if (timer !== null) return
		timer = setIntervalFn(tick, step)
	}

	return {
		animate(key, targetValue, animateOptions = {}) {
			tasks.delete(key)
			const target = Number(targetValue) || 0
			const start = Number(read(key)) || 0
			const round = animateOptions.round === true
			if (start === target) {
				writeBatch({ [key]: target })
				stopTimerIfIdle()
				return false
			}

			tasks.set(key, {
				start,
				target,
				round,
				totalSteps: Math.max(1, Math.round(duration / step)),
				currentStep: 0
			})
			ensureTimer()
			return true
		},
		clear,
		activeKeys() {
			return [...tasks.keys()]
		}
	}
}
