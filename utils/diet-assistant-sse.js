export function callStreamCallback(callback, ...args) {
	if (typeof callback !== 'function') return
	try {
		callback(...args)
	} catch (error) {
		console.warn?.('diet assistant stream callback failed', error)
	}
}

export function normalizeStreamMutation(mutation = null) {
	if (!mutation || typeof mutation !== 'object') return null
	const type = String(mutation.type || '').trim()
	if (!type) return null
	return {
		type,
		recipeId: String(mutation.recipeId || '').trim(),
		recipeTitle: String(mutation.recipeTitle || '').trim(),
		mealType: String(mutation.mealType || '').trim(),
		status: String(mutation.status || '').trim()
	}
}

export function createDietAssistantStreamParser(callbacks = {}) {
	let buffer = ''

	function handleDataLine(line) {
		if (line === '[DONE]') {
			callStreamCallback(callbacks.onDone)
			return
		}

		let event
		try {
			event = JSON.parse(line)
		} catch (error) {
			return
		}

		if (event?.type === 'delta') {
			callStreamCallback(callbacks.onDelta, String(event.delta || ''))
			return
		}
		if (['status', 'tool_start', 'tool_done', 'tool_error'].includes(event?.type)) {
			callStreamCallback(callbacks.onStatus, {
				type: String(event.type || ''),
				message: String(event.message || ''),
				toolName: String(event.toolName || ''),
				mutation: normalizeStreamMutation(event.mutation)
			})
			return
		}
		if (event?.type === 'error') {
			callStreamCallback(callbacks.onError, new Error(event.message || '饮食管家暂时不可用'))
			return
		}
		if (event?.type === 'done') callStreamCallback(callbacks.onDone)
	}

	function handleFrame(frame = '') {
		frame
			.split('\n')
			.map((line) => line.trim())
			.filter((line) => line.startsWith('data:'))
			.map((line) => line.slice(5).trim())
			.filter(Boolean)
			.forEach(handleDataLine)
	}

	return {
		push(text = '') {
			buffer += String(text || '').replace(/\r\n/g, '\n')
			let splitIndex = buffer.indexOf('\n\n')
			while (splitIndex >= 0) {
				const frame = buffer.slice(0, splitIndex)
				buffer = buffer.slice(splitIndex + 2)
				handleFrame(frame)
				splitIndex = buffer.indexOf('\n\n')
			}
		},
		flush() {
			const remaining = buffer.trim()
			buffer = ''
			if (remaining) handleFrame(remaining)
		}
	}
}
