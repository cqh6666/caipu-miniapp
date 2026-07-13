import {
	buildStepCompletedStorageKey,
	createCompletedStepStoragePayload,
	normalizeCompletedStepKeyMap
} from './use-recipe-edit'

export function createActionFeedbackController(options = {}) {
	const {
		onState = () => {},
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
				tick
			})
			timer = setTimeoutFn(() => {
				timer = null
				onState({ visible: false, tick })
			}, Math.max(1200, Number(value?.duration) || 1680))
			return true
		},
		clear() {
			clearTimer()
			onState({ visible: false, tone: '', title: '', description: '', tick })
		},
		status() {
			return { active: timer !== null, tick }
		}
	}
}

export function createStepCompletionController(options = {}) {
	const { storage, onChange = () => {}, onError = () => {} } = options
	let completed = {}

	function publish() {
		completed = { ...completed }
		onChange(completed)
		return completed
	}

	function persist(recipeId) {
		const key = buildStepCompletedStorageKey(recipeId)
		if (!key || !storage) return false
		try {
			if (!Object.keys(completed).length) storage.removeStorageSync(key)
			else storage.setStorageSync(key, createCompletedStepStoragePayload(completed))
			return true
		} catch (error) {
			onError(error)
			return false
		}
	}

	return {
		load(recipeId, stepKeys = []) {
			const key = buildStepCompletedStorageKey(recipeId)
			if (!key || !storage) {
				completed = {}
				return publish()
			}
			try {
				completed = normalizeCompletedStepKeyMap(storage.getStorageSync(key), stepKeys)
			} catch (error) {
				completed = {}
				onError(error)
			}
			return publish()
		},
		toggle(recipeId, stepKeys = [], index = -1) {
			const stepKey = stepKeys[index] || ''
			if (!stepKey) return false
			completed = normalizeCompletedStepKeyMap(completed, stepKeys)
			if (completed[stepKey]) delete completed[stepKey]
			else completed[stepKey] = true
			publish()
			persist(recipeId)
			return true
		},
		reset(recipeId) {
			completed = {}
			publish()
			persist(recipeId)
		},
		persist,
		isCompleted(stepKey) {
			return !!(stepKey && completed[stepKey])
		},
		snapshot() {
			return { ...completed }
		}
	}
}

export function createRecipeLoadController(options = {}) {
	const {
		getCachedRecipe,
		getRecipe,
		getPublicRecipe,
		ensurePrivateAccess = () => {},
		onRecipe = () => {},
		onPublicMeta = () => {},
		onMissing = () => {},
		onError = () => {},
		onState = () => {}
	} = options
	let sequence = 0

	return {
		async load(params = {}) {
			const requestId = ++sequence
			const isCurrent = () => requestId === sequence
			const { recipeId = '', publicViewToken = '', isPublicView = false } = params

			if (isPublicView && publicViewToken) {
				onState({ loading: true })
				try {
					const view = await getPublicRecipe(publicViewToken)
					if (!isCurrent()) return null
					if (!view?.recipe) {
						onMissing({ publicView: true })
						return null
					}
					onPublicMeta(view)
					onRecipe(view.recipe, { source: 'public' })
					return view.recipe
				} catch (error) {
					if (isCurrent()) onError(error, { publicView: true, hasCache: false })
					return null
				} finally {
					if (isCurrent()) onState({ loading: false, resolved: true })
				}
			}

			if (!recipeId) {
				onMissing({ publicView: false })
				onState({ loading: false, resolved: true })
				return null
			}

			ensurePrivateAccess()
			const cached = getCachedRecipe(recipeId)
			if (cached) {
				onRecipe(cached, { source: 'cache' })
				onState({ resolved: true })
			}

			onState({ loading: true })
			try {
				const recipe = await getRecipe(recipeId, { preferCache: !cached })
				if (!isCurrent()) return null
				onRecipe(recipe, { source: 'remote' })
				return recipe
			} catch (error) {
				if (isCurrent() && !cached) onError(error, { publicView: false, hasCache: false })
				return cached || null
			} finally {
				if (isCurrent()) onState({ loading: false, resolved: true })
			}
		},
		cancel() {
			sequence += 1
		}
	}
}
