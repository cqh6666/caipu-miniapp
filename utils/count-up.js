export function easeOutCubic(progress) {
	const normalized = Math.max(0, Math.min(Number(progress) || 0, 1))
	return 1 - Math.pow(1 - normalized, 3)
}

export function createCountUpController(options = {}) {
	const {
		read = () => 0,
		write = () => {},
		duration = 640,
		step = 40,
		setIntervalFn = setInterval,
		clearIntervalFn = clearInterval
	} = options
	const timers = new Map()

	function clear(keys = [...timers.keys()]) {
		keys.forEach((key) => {
			const timer = timers.get(key)
			if (timer !== undefined) clearIntervalFn(timer)
			timers.delete(key)
		})
	}

	return {
		animate(key, targetValue, animateOptions = {}) {
			clear([key])
			const target = Number(targetValue) || 0
			const start = Number(read(key)) || 0
			const round = animateOptions.round === true
			if (start === target) {
				write(key, target)
				return false
			}

			const totalSteps = Math.max(1, Math.round(duration / step))
			let currentStep = 0
			const timer = setIntervalFn(() => {
				currentStep += 1
				const progress = Math.min(1, currentStep / totalSteps)
				const nextValue = start + (target - start) * easeOutCubic(progress)
				write(key, round ? Math.round(nextValue) : nextValue)
				if (progress >= 1) {
					clear([key])
					write(key, target)
				}
			}, step)
			timers.set(key, timer)
			return true
		},
		clear,
		activeKeys() {
			return [...timers.keys()]
		}
	}
}
